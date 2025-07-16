package payment

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	externalservices "github.com/oprimogus/rinha-backend-2025/internal/core/external_services"
	"github.com/oprimogus/rinha-backend-2025/internal/core/money"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/database"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	FindPaymentByID(ctx context.Context, id string) (Payment, error)
	FindProcessorHealth(ctx context.Context, name externalservices.ProcessorName) (externalservices.HealthCheckResponse, error)
	SavePayment(ctx context.Context, payment Payment) error
	SaveProcessorHealthStatus(ctx context.Context, name externalservices.ProcessorName, status externalservices.HealthCheckResponse) error
	GetPaymentsSummary(ctx context.Context, params PaymentSummaryParams) (PaymentSummary, error)
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

	m, err := money.FromStringToFloat(v["amount"])
	if err != nil {
		return Payment{}, err
	}

	t, err := time.Parse(time.RFC3339Nano, v["startedAt"])
	if err != nil {
		return Payment{}, err
	}

	return Payment{
		CorrelationID: id,
		Amount:        m,
		Processor:     v["processor"],
		Status:        PaymentStatus(v["status"]),
		StartedAt:     t,
	}, nil
}

func (r *repository) SavePayment(ctx context.Context, payment Payment) error {
	body := map[string]any{
		"amount":    money.ToCents(payment.Amount),
		"processor": payment.Processor,
		"status":    string(payment.Status),
		"startedAt": payment.StartedAt,
	}

	if err := r.rdb.HSet(ctx, payment.CorrelationID, body).Err(); err != nil {
		return err
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return r.rdb.ZAdd(ctx, "payments", redis.Z{
		Score:  float64(payment.StartedAt.UnixNano()),
		Member: jsonData,
	}).Err()
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

func (r *repository) GetPaymentsSummary(ctx context.Context, params PaymentSummaryParams) (PaymentSummary, error) {
    var results []string
	var err error
    
	if params.Filter {
		min := strconv.FormatInt(params.From.UnixNano(), 10)
		max := strconv.FormatInt(params.To.UnixNano(), 10)
    
		results, err = r.rdb.ZRangeByScore(ctx, "payments", &redis.ZRangeBy{
			Min: min,
			Max: max,
		}).Result()
	} else {
		results, err = r.rdb.ZRange(ctx, "payments", 0, -1).Result()
	}
    
	if err != nil {
		return PaymentSummary{}, err
	}
    
	var defaultCount, fallbackCount int
	var defaultSum, fallbackSum float64
    
	for _, jsonStr := range results {
		var p Payment
    
		if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
			continue
		}
    
		switch p.Processor {
		case string(externalservices.ProcessorDefault):
			defaultCount++
			defaultSum += p.Amount
		case string(externalservices.ProcessorFallback):
			fallbackCount++
			fallbackSum += p.Amount
		}
	}
    
	return PaymentSummary{
		Default: totalPayments{
			TotalRequests: defaultCount,
			TotalAmount:   defaultSum / 100,
		},
		Fallback: totalPayments{
			TotalRequests: fallbackCount,
			TotalAmount:   fallbackSum / 100,
		},
	}, nil
}

