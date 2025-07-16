package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oprimogus/rinha-backend-2025/internal/api"
	"github.com/oprimogus/rinha-backend-2025/internal/config"
	"github.com/oprimogus/rinha-backend-2025/internal/core/payment"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/database"
	logger "github.com/oprimogus/rinha-backend-2025/internal/infra/log"
)

func main() {
	if err := run(); err != nil {
		log.Fatal("deu ruim")
	}
}

func run() error {
	// Config
	cfg := config.GetInstance()
	
	// Config logger
	logger.InitLogger(os.Stdout)
	slog.Info("Starting application...")
	
	// Database
	db := database.GetRedis()
	
	workerCount := 20
	slog.Info("Worker configuration", "count", workerCount)
	

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Workers
	repo := payment.NewRepository(db)
	paymentWorker := payment.NewPaymentWorker(repo, workerCount)
	
	// Inicia o worker em background
	go func() {
		slog.Info("Starting payment worker...")
		paymentWorker.Run(ctx, workerCount)
	}()
	
	// Inicializa o servidor HTTP
	handler := api.InitRouter(db)
	srv := &http.Server{
		Addr:         ":" + cfg.API.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Inicia o servidor em background
	go func() {
		slog.Info("Starting HTTP server", "port", cfg.API.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			cancel() // Cancela o contexto se o servidor falhar
		}
	}()
	
	// Canal para capturar sinais de shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// Aguarda sinal de shutdown ou erro no contexto
	select {
	case sig := <-quit:
		slog.Info("Received shutdown signal", "signal", sig.String())
	case <-ctx.Done():
		slog.Info("Context cancelled, shutting down...")
	}
	
	// Graceful shutdown
	return gracefulShutdown(srv, paymentWorker, cancel)
}

func gracefulShutdown(srv *http.Server, paymentWorker *payment.PaymentWorker, cancel context.CancelFunc) error {
	slog.Info("Starting graceful shutdown...")
	
	// Timeout total para shutdown
	shutdownTimeout := 30 * time.Second
	ctx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	
	// Canal para sinalizar que o shutdown foi concluído
	done := make(chan error, 1)
	
	go func() {
		defer close(done)
		
		// 1. Para de aceitar novas conexões HTTP
		slog.Info("Shutting down HTTP server...")
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("HTTP server shutdown failed", "error", err)
			done <- err
			return
		}
		slog.Info("HTTP server shutdown completed")
		
		// 2. Para o payment worker
		slog.Info("Shutting down payment worker...")
		if err := paymentWorker.Shutdown(ctx); err != nil {
			slog.Error("Payment worker shutdown failed", "error", err)
			done <- err
			return
		}
		slog.Info("Payment worker shutdown completed")
		
		// 3. Cancela o contexto principal
		cancel()
		
		slog.Info("Graceful shutdown completed successfully")
		done <- nil
	}()
	
	// Aguarda o shutdown completar ou timeout
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		slog.Error("Shutdown timeout exceeded", "timeout", shutdownTimeout)
		return ctx.Err()
	}
}