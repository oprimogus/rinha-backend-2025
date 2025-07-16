package externalservices

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/oprimogus/rinha-backend-2025/internal/config"
)

type PaymentProcessor interface {
	ProcessorName() ProcessorName
	ProcessPayment(ctx context.Context, params PaymentParams) (PaymentResponse, error)
	VerifyHealth() (HealthCheckResponse, error)
}

type ProcessorName string

const (
	ProcessorDefault  ProcessorName = "default"
	ProcessorFallback ProcessorName = "fallback"
)

type BasePaymentProcessorService struct {
	Name    ProcessorName
	BaseURL string
	Client  *http.Client
}

func (b *BasePaymentProcessorService) ProcessorName() ProcessorName {
	return b.Name
}

func (b *BasePaymentProcessorService) ProcessPayment(ctx context.Context, params PaymentParams) (PaymentResponse, error) {
	slog.InfoContext(ctx, "processing payment with "+string(b.Name), "correlationID", params.CorrelationID)
	url := strings.Join([]string{b.BaseURL, "/payments"}, "")

	payload, err := json.Marshal(params)
	if err != nil {
		return PaymentResponse{}, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return PaymentResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.Client.Do(req)
	if err != nil {
		return PaymentResponse{}, err
	}
	defer resp.Body.Close()

	slog.Info("process payment response", "response", resp.StatusCode)

	var response PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return PaymentResponse{}, err
	}
	slog.InfoContext(ctx, "payment processed with "+string(b.Name), "correlationID", params.CorrelationID, "response", response)
	return response, nil
}

func (b *BasePaymentProcessorService) VerifyHealth() (HealthCheckResponse, error) {
	url := strings.Join([]string{b.BaseURL, "/payments/service-health"}, "")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return HealthCheckResponse{}, err
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return HealthCheckResponse{}, err
	}
	defer resp.Body.Close()

	var response HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return HealthCheckResponse{}, err
	}

	return response, nil
}

type DefaultPaymentProcessor struct {
	*BasePaymentProcessorService
}

func NewDefaultPaymentProcessor() *DefaultPaymentProcessor {
	cfg := config.GetInstance()
	return &DefaultPaymentProcessor{&BasePaymentProcessorService{
		Name:    ProcessorDefault,
		BaseURL: cfg.ExternalServices.DefaultPaymentProcessor.BaseURL,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}}
}

type FallbackPaymentProcessor struct {
	*BasePaymentProcessorService
}

func NewFallbackPaymentProcessor() *FallbackPaymentProcessor {
	cfg := config.GetInstance()
	return &FallbackPaymentProcessor{&BasePaymentProcessorService{
		Name:    ProcessorFallback,
		BaseURL: cfg.ExternalServices.FallbackPaymentProcessor.BaseURL,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}}
}

func FindPaymentProcessorStrategy(name ProcessorName) PaymentProcessor {
	switch name {
	case ProcessorDefault:
		return NewDefaultPaymentProcessor()
	case ProcessorFallback:
		return NewFallbackPaymentProcessor()
	default:
		return nil
	}
}
