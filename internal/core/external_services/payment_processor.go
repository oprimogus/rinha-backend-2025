package externalservices

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/oprimogus/rinha-backend-2025/internal/config"
)

type PaymentProcessor interface {
	ProcessPayment(params PaymentParams) (PaymentResponse, error)
	VerifyHealth() (HealthCheckResponse, error)
}

type BasePaymentProcessorService struct {
	Name    string
	BaseURL string
	Client  *http.Client
}

func (b *BasePaymentProcessorService) ProcessPayment(params PaymentParams) (PaymentResponse, error) {
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
	slog.Info("Creating Default Payment Processor: Config: ", slog.Any("config", cfg))
	return &DefaultPaymentProcessor{&BasePaymentProcessorService{
		Name:    "Default Payment Processor",
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
		Name:    "Fallback Payment Processor",
		BaseURL: cfg.ExternalServices.FallbackPaymentProcessor.BaseURL,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}}
}
