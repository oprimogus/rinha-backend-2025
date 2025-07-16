package payment

import (
	"context"
	"log/slog"
	"time"

	externalservices "github.com/oprimogus/rinha-backend-2025/internal/core/external_services"
)

type Service struct {
	r                 Repository
	defaultProcessor  externalservices.PaymentProcessor
	fallbackProcessor externalservices.PaymentProcessor
}

func NewService(r Repository) *Service {
	return &Service{
		r:                 r,
		defaultProcessor:  externalservices.NewDefaultPaymentProcessor(),
		fallbackProcessor: externalservices.NewFallbackPaymentProcessor(),
	}
}

func (s *Service) GetHealthStatus(ctx context.Context, name externalservices.ProcessorName) (externalservices.HealthCheckResponse, error) {
	p := externalservices.FindPaymentProcessorStrategy(name)

	h, err := p.VerifyHealth()
	if err != nil {
		slog.Info("fail on get health check status of default processor", "error", err)
		return externalservices.HealthCheckResponse{}, err
	}
	err = s.r.SaveProcessorHealthStatus(ctx, p.ProcessorName(), h)
	if err != nil {
		slog.Info("fail on save health check status of default processor", "error", err)
		return externalservices.HealthCheckResponse{}, err
	}
	return h, nil
}

func (s *Service) GetPaymentsSummary(ctx context.Context, params PaymentSummaryParams) (PaymentSummary, error) {
	return s.r.GetPaymentsSummary(ctx, params)
}

func (s *Service) ProcessPayment(ctx context.Context, params PaymentParams) (Payment, error) {
	payment := Payment{
		CorrelationID: params.CorrelationID,
		Amount:        params.Amount,
		Status:        PaymentStatusPending,
		StartedAt:     time.Now(),
	}

	// Salva o pagamento primeiro
	err := s.r.SavePayment(ctx, payment)
	if err != nil {
		slog.Error("fail on save payment", "error", err, "payload", payment)
		return Payment{}, err
	}

	if !s.sendToQueueWithRetry(ctx, payment, 3) {
		go func() {
			slog.Error("failed to queue payment after retries", "payment", payment)

			payment.Status = PaymentStatusFailed
			if updateErr := s.r.SavePayment(ctx, payment); updateErr != nil {
				slog.Error("failed to update payment status", "error", updateErr, "payment", payment)
			}
		}()

		return payment, nil
	}

	return payment, nil
}

func (s *Service) sendToQueueWithRetry(ctx context.Context, payment Payment, maxRetries int) bool {
	for i := range maxRetries {
		if SendToQueue(payment) {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Millisecond * 100 * time.Duration(i+1)): // Backoff exponencial
			continue
		}
	}
	return false
}

func (s *Service) ProcessPaymentAsync(ctx context.Context, p Payment) error {
	var hDefault, hFallback externalservices.HealthCheckResponse
	var hDefaultErr, hFallbackErr error

	hDefault, hDefaultErr = s.r.FindProcessorHealth(ctx, s.defaultProcessor.ProcessorName())
	hFallback, hFallbackErr = s.r.FindProcessorHealth(ctx, s.fallbackProcessor.ProcessorName())

	if hDefaultErr != nil && hFallbackErr != nil {
		slog.Error("processors are down")
		return ErrAllProcessorsAreDown
	}
	if hDefaultErr != nil {
		slog.Error("fail on get health check status of default processor", "error", hDefaultErr)
		return s.processPaymentWithFallback(ctx, p)
	}
	if hFallbackErr != nil {
		slog.Error("fail on get health check status of fallback processor", "error", hFallbackErr)
		return s.processPaymentWithDefault(ctx, p)
	}

	if hDefault.Failing && hFallback.Failing {
		return ErrAllProcessorsAreDown
	}
	if hDefault.Failing {
		return s.processPaymentWithFallback(ctx, p)
	}
	if hFallback.Failing {
		return s.processPaymentWithDefault(ctx, p)
	}
	if !hDefault.Failing && !hFallback.Failing {
		if hDefault.MinResponseTime > 5*1000 {
			return s.processPaymentWithFallback(ctx, p)
		}
		return s.processPaymentWithDefault(ctx, p)
	}
	return nil
}

func (s *Service) processPaymentWithDefault(ctx context.Context, p Payment) error {
	resp, err := s.defaultProcessor.ProcessPayment(ctx, externalservices.PaymentParams{
		CorrelationID: p.CorrelationID,
		Amount:        p.Amount,
		RequestedAt:   p.StartedAt,
	})

	p.Processor = string(s.defaultProcessor.ProcessorName())
	if err != nil {
		p.Status = PaymentStatusFailed
		slog.Error("failed to process payment with default processor",
			"error", err,
			"correlation_id", p.CorrelationID,
			"response", resp,
		)
	} else {
		p.Status = PaymentStatusSuccess
		slog.Info("payment processed with default processor", "correlation_id", p.CorrelationID)
	}

	if saveErr := s.r.SavePayment(ctx, p); saveErr != nil {
		slog.Error("failed to save payment",
			"error", saveErr,
			"correlation_id", p.CorrelationID,
			"status", p.Status,
		)
		return saveErr
	}

	return err
}

func (s *Service) processPaymentWithFallback(ctx context.Context, p Payment) error {
	resp, err := s.fallbackProcessor.ProcessPayment(ctx, externalservices.PaymentParams{
		CorrelationID: p.CorrelationID,
		Amount:        p.Amount,
		RequestedAt:   p.StartedAt,
	})

	p.Processor = string(s.fallbackProcessor.ProcessorName())

	if err != nil {
		slog.Error("failed to process payment with fallback processor",
			"error", err,
			"correlation_id", p.CorrelationID,
			"response", resp,
		)
		return err
	}

	p.Status = PaymentStatusSuccess
	slog.Info("payment processed with fallback processor", "correlation_id", p.CorrelationID)

	if saveErr := s.r.SavePayment(ctx, p); saveErr != nil {
		slog.Error("failed to save payment after fallback processing",
			"error", saveErr,
			"correlation_id", p.CorrelationID,
			"status", p.Status,
		)
		return saveErr
	}

	return nil
}
