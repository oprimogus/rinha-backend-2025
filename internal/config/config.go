package config

import (
	"log/slog"
	"os"

	"github.com/subosito/gotenv"
)

var cfg *Config

type Config struct {
    API API
    ExternalServices ExternalServices
}

type API struct {
    Port string
    BasePath string
}

type ExternalServices struct {
    DefaultPaymentProcessor ExternalService
    FallbackPaymentProcessor ExternalService
}

type ExternalService struct {
    BaseURL string
}

func newConfig() *Config {
    err := gotenv.Load()
	if err != nil {
		slog.Info("arquivo .env não encontrado, usando variáveis de ambiente")
	}
	
    return &Config{
        API: API{
            Port: os.Getenv("API_PORT"),
            BasePath: os.Getenv("API_BASE_PATH"),
        },
        ExternalServices: ExternalServices{
            DefaultPaymentProcessor: ExternalService{
                BaseURL: os.Getenv("EXTERNAL_SERVICE_DEFAULT_PAYMENT_PROCESSOR_URL"),
            },
            FallbackPaymentProcessor: ExternalService{
                BaseURL: os.Getenv("EXTERNAL_SERVICE_FALLBACK_PAYMENT_PROCESSOR_URL"),
            },
        },
    }
}

func GetInstance() *Config {
	if cfg == nil {
		cfg = newConfig()
	}
	return cfg
}