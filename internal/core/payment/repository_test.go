package payment_test

import (
	"context"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oprimogus/rinha-backend-2025/internal/config"
	externalservices "github.com/oprimogus/rinha-backend-2025/internal/core/external_services"
	"github.com/oprimogus/rinha-backend-2025/internal/core/money"
	"github.com/oprimogus/rinha-backend-2025/internal/core/payment"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/database"
	"github.com/oprimogus/rinha-backend-2025/internal/infra/testcontainers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RepositoryTestSuite struct {
    suite.Suite
    mockRedis *testcontainers.Container
    db        *database.Redis
    r payment.Repository
}

func (s *RepositoryTestSuite) SetupSuite() {
    ctx := context.Background()
    mockRedis, err := testcontainers.MakeRedis(ctx)
    
    if err != nil {
        assert.Error(s.T(), err)
    }
    
    s.mockRedis = mockRedis
    
    redisPort, err := strconv.Atoi(mockRedis.Port)
    if err != nil {
        assert.Error(s.T(), err)
    }
    
    cfg := config.GetInstance()
    cfg.Redis.Host = "localhost"
    cfg.Redis.Port = redisPort
    cfg.Redis.Password = ""
    
    db := database.GetRedis()
    s.db = db
    s.r = payment.NewRepository(db)
}

func (s *RepositoryTestSuite) TearDownSuite() {
	ctx := context.Background()
	s.mockRedis.Kill(ctx)
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

func (s *RepositoryTestSuite) TestFindPaymentByID() {
    ctx := context.Background()
    id, err := uuid.NewV7()
    assert.NoError(s.T(), err)
    
    t := time.Now().Truncate(time.Second)
    
    p := payment.Payment{
        CorrelationID: id.String(),
        Amount: 100.00,
        StartedAt: t,
        Processor: string(externalservices.ProcessorDefault),
    }
    
    slog.Info("payment generated and sended to db", "body", p)
    
    err = s.db.HSet(ctx, p.CorrelationID, map[string]any{
        "amount":        money.ToCents(p.Amount),
        "processor":     p.Processor,
        "startedAt":     p.StartedAt.Format(time.RFC3339),
    }).Err()
    assert.NoError(s.T(), err)
    
    payment, err := s.r.FindPaymentByID(ctx, p.CorrelationID)
    slog.Info("payment retrieved", "body", payment)
    assert.NoError(s.T(), err)
    assert.Equal(s.T(), p.CorrelationID, payment.CorrelationID)
    assert.Equal(s.T(), p.Amount, payment.Amount)
    assert.Equal(s.T(), p.StartedAt, payment.StartedAt)
    assert.Equal(s.T(), p.Processor, payment.Processor)
}

func (s *RepositoryTestSuite) TestSavePayment() {
    ctx := context.Background()
    id, err := uuid.NewV7()
    assert.NoError(s.T(), err)
    
    t := time.Now().Truncate(time.Second)
    
    p := payment.Payment{
        CorrelationID: id.String(),
        Amount: 1000.00,
        StartedAt: t,
        Processor: string(externalservices.ProcessorDefault),
    }
    
    slog.Info("payment generated and sended to db", "body", p)
    
    err = s.r.SavePayment(ctx, p)
    assert.NoError(s.T(), err)
    
    payment, err := s.r.FindPaymentByID(ctx, p.CorrelationID)
    slog.Info("payment retrieved", "body", payment)
    assert.NoError(s.T(), err)
    assert.Equal(s.T(), p.CorrelationID, payment.CorrelationID)
    assert.Equal(s.T(), p.Amount, payment.Amount)
    assert.Equal(s.T(), p.StartedAt, payment.StartedAt)
    assert.Equal(s.T(), p.Processor, payment.Processor)
}

func (s *RepositoryTestSuite) TestFindProcessorHealth() {
    ctx := context.Background()
    health := externalservices.HealthCheckResponse{
        Failing: false,
        MinResponseTime: 15,
    }
    err := s.db.HSet(ctx, string(externalservices.ProcessorDefault), map[string]any{
		"failing":         health.Failing,
		"minResponseTime": health.MinResponseTime,
	}).Err()
    assert.NoError(s.T(), err)
    
    h, err := s.r.FindProcessorHealth(ctx, externalservices.ProcessorDefault)
    assert.NoError(s.T(), err)
    assert.Equal(s.T(), health.Failing, h.Failing)
    assert.Equal(s.T(), health.MinResponseTime, h.MinResponseTime)
}

func (s *RepositoryTestSuite) TestSaveProcessorHealthStatus() {
    ctx := context.Background()
    processor := externalservices.ProcessorDefault
    health := externalservices.HealthCheckResponse{
        Failing: false,
        MinResponseTime: 15,
    }
    
    err := s.r.SaveProcessorHealthStatus(ctx, processor, health)
    assert.NoError(s.T(), err)
    
    h, err := s.r.FindProcessorHealth(ctx, processor)
    assert.NoError(s.T(), err)
    assert.Equal(s.T(), health.Failing, h.Failing)
    assert.Equal(s.T(), health.MinResponseTime, h.MinResponseTime)
}

func (s *RepositoryTestSuite) TestGetPaymentsSummary_WithFilter() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	payments := []payment.Payment{
		{
			CorrelationID: uuid.New().String(),
			Amount:        100.00,
			Processor:     string(externalservices.ProcessorDefault),
			StartedAt:     now.Add(-10 * time.Second),
		},
		{
			CorrelationID: uuid.New().String(),
			Amount:        200.00,
			Processor:     string(externalservices.ProcessorFallback),
			StartedAt:     now.Add(-5 * time.Second),
		},
		{
			CorrelationID: uuid.New().String(),
			Amount:        50.00,
			Processor:     string(externalservices.ProcessorDefault),
			StartedAt:     now.Add(-2 * time.Second),
		},
	}

	for _, p := range payments {
		err := s.r.SavePayment(ctx, p)
		assert.NoError(s.T(), err)
	}

	filter := payment.PaymentSummaryParams{
		Filter: true,
		From:   now.Add(-15 * time.Second),
		To:     now,
	}

	summary, err := s.r.GetPaymentsSummary(ctx, filter)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), 2, summary.Default.TotalRequests)
	assert.InDelta(s.T(), 150.00, summary.Default.TotalAmount, 0.01)

	assert.Equal(s.T(), 1, summary.Fallback.TotalRequests)
	assert.InDelta(s.T(), 200.00, summary.Fallback.TotalAmount, 0.01)
}

func (s *RepositoryTestSuite) TestGetPaymentsSummary_WithoutFilter() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	payments := []payment.Payment{
		{
			CorrelationID: uuid.New().String(),
			Amount:        300.00,
			Processor:     string(externalservices.ProcessorFallback),
			StartedAt:     now.Add(-30 * time.Second),
		},
		{
			CorrelationID: uuid.New().String(),
			Amount:        100.00,
			Processor:     string(externalservices.ProcessorDefault),
			StartedAt:     now.Add(-25 * time.Second),
		},
	}

	for _, p := range payments {
		err := s.r.SavePayment(ctx, p)
		assert.NoError(s.T(), err)
	}

	filter := payment.PaymentSummaryParams{Filter: false}

	summary, err := s.r.GetPaymentsSummary(ctx, filter)
	assert.NoError(s.T(), err)

	assert.GreaterOrEqual(s.T(), summary.Default.TotalRequests, 1)
	assert.GreaterOrEqual(s.T(), summary.Fallback.TotalRequests, 1)

	assert.GreaterOrEqual(s.T(), summary.Default.TotalAmount, 1.0)
	assert.GreaterOrEqual(s.T(), summary.Fallback.TotalAmount, 1.0)
}


