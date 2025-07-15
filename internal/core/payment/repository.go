package payment

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"

	externalservices "github.com/oprimogus/rinha-backend-2025/internal/core/external_services"
	"github.com/oprimogus/rinha-backend-2025/internal/core/money"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/database"
)

type Repository interface {
	FindPaymentByID(ctx context.Context, id string) (Payment, error)
	FindProcessorHealth(ctx context.Context, name externalservices.ProcessorName) (externalservices.HealthCheckResponse, error)
	SavePayment(ctx context.Context, payment Payment) error
	SaveProcessorHealthStatus(ctx context.Context, name externalservices.ProcessorName, status externalservices.HealthCheckResponse) error
}

type repository struct {
	rdb *database.Redis
}

func NewRepository(redis *database.Redis) Repository {
	return &repository{
		rdb: redis,
	}
}

func (r *repository) FindPaymentByID(ctx context.Context, id string) (Payment, error) {
	v, err := r.rdb.HGetAll(ctx, id).Result()
	if err != nil {
		return Payment{}, err
	}

	if len(v) == 0 {
		return Payment{}, errors.New("payment not found")
	}
	
	slog.Info("map retrieved", "body", v)

	m, err := money.FromStringToFloat(v["amount"])
	if err != nil {
		return Payment{}, err
	}

	t, err := time.Parse(time.RFC3339, v["startedAt"])
	if err != nil {
		return Payment{}, err
	}

	return Payment{
		CorrelationID: id,
		Amount:        m,
		Processor:     v["processor"],
		StartedAt:     t,
	}, nil
}

func (r *repository) FindProcessorHealth(ctx context.Context, name externalservices.ProcessorName) (externalservices.HealthCheckResponse, error) {
	v, err := r.rdb.HGetAll(ctx, string(name)).Result()
	if err != nil {
		return externalservices.HealthCheckResponse{}, err
	}

	if len(v) == 0 {
		return externalservices.HealthCheckResponse{}, errors.New("health status not found")
	}

	minResponseTime, err := strconv.Atoi(v["minResponseTime"])
	if err != nil {
		return externalservices.HealthCheckResponse{}, err
	}

	failing, err := strconv.ParseBool(v["failing"])
	if err != nil {
		return externalservices.HealthCheckResponse{}, err
	}

	return externalservices.HealthCheckResponse{
		Failing:         failing,
		MinResponseTime: minResponseTime,
	}, nil
}

func (r *repository) SavePayment(ctx context.Context, payment Payment) error {
	err := r.rdb.HSet(ctx, payment.CorrelationID, map[string]any{
		"amount":    money.ToCents(payment.Amount),
		"processor": payment.Processor,
		"startedAt": payment.StartedAt.Truncate(time.Second).Format(time.RFC3339),
	}).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) SaveProcessorHealthStatus(ctx context.Context, name externalservices.ProcessorName, health externalservices.HealthCheckResponse) error {
	err := r.rdb.HSet(ctx, string(name), map[string]any{
		"failing":         health.Failing,
		"minResponseTime": health.MinResponseTime,
	}).Err()
	if err != nil {
		return err
	}

	return nil
}
