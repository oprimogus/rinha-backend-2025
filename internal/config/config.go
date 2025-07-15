package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/subosito/gotenv"
)

var cfg *Config

type Config struct {
    API API
    Redis Redis
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

type Redis struct {
    Host string
    Port int
    Password string
}

func newConfig() *Config {
    err := gotenv.Load()
	if err != nil {
		slog.Info("arquivo .env não encontrado, usando variáveis de ambiente")
	}
	
	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		slog.Error("erro ao converter REDIS_PORT para int", slog.Any("err", err), slog.Any("value", os.Getenv("REDIS_PORT")))
		redisPort = 0
	}

    return &Config{
        API: API{
            Port: os.Getenv("API_PORT"),
            BasePath: os.Getenv("API_BASE_PATH"),
        },
        Redis: Redis{
            Host: os.Getenv("REDIS_HOST"),
            Port: redisPort,
            Password: os.Getenv("REDIS_PASSWORD"),
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