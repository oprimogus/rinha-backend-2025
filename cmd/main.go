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
    logger.InitLogger(os.Stdout, slog.LevelInfo)
    
    // Start web server
    handler := api.InitRouter()
    
    srv := &http.Server{
        Addr:    ":" + cfg.API.Port,
        Handler: handler,
    }
    
    go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()
    
    // Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")
   
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	<- ctx.Done()
	slog.Info("timeout of 5 seconds")
   
	slog.Info("Server exiting")
	err := srv.Shutdown(context.Background())
   
	return err
}