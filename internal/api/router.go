package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oprimogus/rinha-backend-2025/internal/api/middlewares"
	"github.com/oprimogus/rinha-backend-2025/internal/config"
	"github.com/oprimogus/rinha-backend-2025/internal/core/payment"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/database"
	logger "github.com/oprimogus/rinha-backend-2025/internal/infra/log"
)

func InitRouter(db *database.Redis) http.Handler {
	cfg := config.GetInstance()
	r := chi.NewRouter()
	r.Use(logger.LoggingMiddleware)
	r.Use(middlewares.JSON)
	r.Use(middleware.Recoverer)
	
	payment.SetupRoutes(r, db)

	slog.Info(fmt.Sprintf("Docs available in http://localhost:%s%s/docs", cfg.API.Port, cfg.API.BasePath))
	slog.Info(fmt.Sprintf("Listening and serving in 0.0.0.0:%v", cfg.API.Port))

	return r
}
