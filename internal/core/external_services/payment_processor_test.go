package externalservices_test

import (
	// "log/slog"
	"testing"

	// "github.com/google/uuid"
	"github.com/oprimogus/rinha-backend-2025/internal/config"
	// externalservices "github.com/oprimogus/rinha-backend-2025/internal/core/external_services"
	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ProcessorsTestSuite struct {
    suite.Suite
}

func (s *ProcessorsTestSuite) SetupSuite() {
    cfg := config.GetInstance()
    cfg.ExternalServices.DefaultPaymentProcessor.BaseURL = "http://localhost:8001"
    cfg.ExternalServices.FallbackPaymentProcessor.BaseURL = "http://localhost:8002"
}

// func (s *ProcessorsTestSuite) TestHealthCheckDefaultProcessor() {
//     df := externalservices.NewDefaultPaymentProcessor()
    
//     hc, err := df.VerifyHealth()
//     if err != nil {
//         slog.Info(err.Error())
//         s.Fail("Error verifying health", err)
//     }
    
//     assert.Equal(s.T(),  hc.Failing, false)
//     assert.Equal(s.T(),  hc.MinResponseTime, 0)
// }

// func (s *ProcessorsTestSuite) TestHealthCheckFallbackProcessor() {
//     fb := externalservices.NewFallbackPaymentProcessor()
    
//     hc, err := fb.VerifyHealth()
//     if err != nil {
//         slog.Info(err.Error())
//         s.Fail("Error verifying health", err)
//     }
    
//     assert.Equal(s.T(),  hc.Failing, false)
//     assert.Equal(s.T(),  hc.MinResponseTime, 0)
// }

// func (s *ProcessorsTestSuite) TestPaymentOnDefaultProcessor() {
//     df := externalservices.NewDefaultPaymentProcessor()
    
//     id, err := uuid.NewV7()
//     if err != nil {
//         slog.Info(err.Error())
//         s.Fail("Error generating UUID", err)
//     }
    
//     resp, err := df.ProcessPayment(externalservices.PaymentParams{
//         CorrelationID: id.String(),
//         Amount: 100,
//     })
    
//     if err != nil {
//         slog.Info(err.Error())
//         s.Fail("Error processing payment", err)
//     }
    
//     slog.Info("Processed payment", "body", resp)
// }

// func (s *ProcessorsTestSuite) TestPaymentOnFallbackProcessor() {
//     fb := externalservices.NewFallbackPaymentProcessor()
    
//     id, err := uuid.NewV7()
//     if err != nil {
//         slog.Info(err.Error())
//         s.Fail("Error generating UUID", err)
//     }
    
//     resp, err := fb.ProcessPayment(externalservices.PaymentParams{
//         CorrelationID: id.String(),
//         Amount: 100,
//     })
    
//     if err != nil {
//         slog.Info(err.Error())
//         s.Fail("Error processing payment", err)
//     }
    
//     slog.Info("Processed payment", "body", resp)
// }

func TestProcessorsTestSuite(t *testing.T) {
    suite.Run(t, new(ProcessorsTestSuite))
}
