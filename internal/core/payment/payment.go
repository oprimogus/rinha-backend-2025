package payment

import (
	"time"
)

type Payment struct {
    CorrelationID string `json:"correlationId"`
    Amount        float64 `json:"amount"`
    Processor     string `json:"processor"`
    StartedAt     time.Time `json:"startedAt"`
}

type totalPayments struct {
    TotalRequests int `json:"totalRequests"`
    TotalAmount float64 `json:"totalAmount"`
}

type PaymentSummaryParams struct {
    From time.Time `json:"from"`
    To time.Time `json:"to"`
}

type PaymentSummary struct {
    Default totalPayments `json:"default"`
    Fallback totalPayments `json:"fallback"`
}
