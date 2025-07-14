package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oprimogus/rinha-backend-2025/internal/api/middlewares"
	"github.com/oprimogus/rinha-backend-2025/internal/config"
)

func InitRouter() http.Handler {
    cfg := config.GetInstance()
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middlewares.JSON)
    r.Use(middleware.Recoverer)
    
    
    slog.Info(fmt.Sprintf("Docs available in http://localhost:%s%s/docs", cfg.API.Port, cfg.API.BasePath))
	slog.Info(fmt.Sprintf("Listening and serving in 0.0.0.0:%v", cfg.API.Port))
    
    return r   
}