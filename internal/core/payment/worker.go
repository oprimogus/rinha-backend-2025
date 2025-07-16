package payment

import (
	"context"
	"log/slog"
	"sync"
	"time"

	externalservices "github.com/oprimogus/rinha-backend-2025/internal/core/external_services"
	"golang.org/x/sync/errgroup"
)

var (
	paymentQueue    = make(chan Payment, 10000)
	paymentErrQueue = make(chan Payment, 1000)
	paymentPool     = sync.Pool{
		New: func() any {
			return &Payment{}
		},
	}
)

func SendToQueue(payment Payment) bool {
	select {
	case paymentQueue <- payment:
		return true
	default:
		// Do back pressure
		slog.Warn("payment queue is full, dropping message")
		return false
	}
}

func ReprocessPayment(payment Payment) bool {
	select {
	case paymentErrQueue <- payment:
		return true
	default:
		slog.Error("error queue is full, dropping failed payment")
		return false
	}
}

type PaymentWorker struct {
	r       Repository
	service *Service

	workerCount int
	wg          sync.WaitGroup

	processed  int64
	failed     int64
	metricsMux sync.RWMutex

	rateLimiter chan struct{}
}

func NewPaymentWorker(repository Repository, workerCount int) *PaymentWorker {
	return &PaymentWorker{
		r:           repository,
		service:     NewService(repository),
		workerCount: workerCount,
		rateLimiter: make(chan struct{}, workerCount*2),
	}
}

func (w *PaymentWorker) Run(ctx context.Context, workers int) {
	go w.StartHealthCheckJob(ctx, 8*time.Second)
	
	w.StartProcessPaymentsWorker(ctx)
	
	go w.StartErrorReprocessingWorker(ctx)
	
	go w.StartMetricsWorker(ctx)
}

func (w *PaymentWorker) StartHealthCheckJob(ctx context.Context, interval time.Duration) {
	slog.Info("Starting health check job...")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Finalizing health check job...")
			return
		case <-ticker.C:
			go func() {
				ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				
				if err := w.GetHealthStatus(ctxTimeout); err != nil {
					slog.Warn("health check failed", "error", err)
				}
			}()
		}
	}
}

func (w *PaymentWorker) GetHealthStatus(ctx context.Context) error {
	slog.InfoContext(ctx, "Searching health status")
	
	g, ctx := errgroup.WithContext(ctx)
	
	// Processa health checks em paralelo
	g.Go(func() error {
		return w.checkProcessorHealth(ctx, externalservices.ProcessorDefault)
	})
	
	g.Go(func() error {
		return w.checkProcessorHealth(ctx, externalservices.ProcessorFallback)
	})
	
	if err := g.Wait(); err != nil {
		slog.Error("fail on get health check", "error", err)
		return err
	}
	
	return nil
}

func (w *PaymentWorker) checkProcessorHealth(ctx context.Context, processor externalservices.ProcessorName) error {
	h, err := w.service.GetHealthStatus(ctx, processor)
	if err != nil {
		slog.Info("fail on get health check status", "processor", processor, "error", err)
		return err
	}
	
	err = w.r.SaveProcessorHealthStatus(ctx, processor, h)
	if err != nil {
		slog.Info("fail on save health check status", "processor", processor, "error", err)
		return err
	}
	
	return nil
}

func (w *PaymentWorker) StartProcessPaymentsWorker(ctx context.Context) {
	slog.Info("Starting process payment workers", "count", w.workerCount)
	
	for i := range w.workerCount {
		w.wg.Add(1)
		go w.paymentWorker(ctx, i)
	}
}

func (w *PaymentWorker) paymentWorker(ctx context.Context, workerID int) {
	defer w.wg.Done()
	
	slog.Info("Payment worker started", "worker", workerID)
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Payment worker stopped", "worker", workerID)
			return
			
		case payment := <-paymentQueue:
			// Rate limiting
			w.rateLimiter <- struct{}{}
			
			// Processa pagamento
			if err := w.processPaymentWithRetry(ctx, payment, workerID); err != nil {
				w.incrementFailed()
				if !ReprocessPayment(payment) {
					slog.Error("Failed to requeue payment", "worker", workerID, "payment", payment)
				}
			} else {
				w.incrementProcessed()
			}
			<-w.rateLimiter
		}
	}
}

func (w *PaymentWorker) processPaymentWithRetry(ctx context.Context, payment Payment, workerID int) error {
	// Timeout específico para cada processamento
	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	slog.Info("Processing payment", "worker", workerID, "payment", payment)
	
	err := w.service.ProcessPaymentAsync(ctxTimeout, payment)
	if err != nil {
		slog.Error("Failed to process payment", "error", err, "worker", workerID)
		return err
	}
	
	slog.Info("Payment processed successfully", "worker", workerID, "payment", payment)
	return nil
}

func (w *PaymentWorker) StartErrorReprocessingWorker(ctx context.Context) {
	slog.Info("Starting error reprocessing worker...")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				slog.Info("Error reprocessing worker stopped")
				return
				
			case <-ticker.C:
				w.processBatchErrors(ctx)
				
			case payment := <-paymentErrQueue:
				time.Sleep(5 * time.Second)
				
				if !SendToQueue(payment) {
					slog.Error("Failed to requeue payment from error queue")
				}
			}
		}
	}()
}

func (w *PaymentWorker) processBatchErrors(ctx context.Context) {
	processed := 0
	maxBatch := 100
	
	for processed < maxBatch {
		select {
		case payment := <-paymentErrQueue:
			time.Sleep(1 * time.Second)
			
			if !SendToQueue(payment) {
				select {
				case paymentErrQueue <- payment:
				default:
					slog.Error("Both queues are full, dropping payment")
				}
			}
			processed++
			
		default:
			return
		}
	}
}

func (w *PaymentWorker) StartMetricsWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.logMetrics()
		}
	}
}

func (w *PaymentWorker) logMetrics() {
	w.metricsMux.RLock()
	processed := w.processed
	failed := w.failed
	w.metricsMux.RUnlock()
	
	slog.Info("Worker metrics",
		"processed", processed,
		"failed", failed,
		"queue_size", len(paymentQueue),
		"error_queue_size", len(paymentErrQueue),
	)
}

func (w *PaymentWorker) incrementProcessed() {
	w.metricsMux.Lock()
	w.processed++
	w.metricsMux.Unlock()
}

func (w *PaymentWorker) incrementFailed() {
	w.metricsMux.Lock()
	w.failed++
	w.metricsMux.Unlock()
}

func (w *PaymentWorker) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down payment worker...")
	
	// Fecha os canais para sinalizar parada
	close(paymentQueue)
	close(paymentErrQueue)
	
	// Aguarda todos os workers terminarem
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	
	// Aguarda com timeout
	select {
	case <-done:
		slog.Info("Payment worker shutdown completed")
		return nil
	case <-ctx.Done():
		slog.Warn("Payment worker shutdown timeout")
		return ctx.Err()
	}
}
