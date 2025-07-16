package payment

import (
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSuccess PaymentStatus = "success"
	PaymentStatusFailed  PaymentStatus = "failed"
)

type Payment struct {
    CorrelationID string `json:"correlationId"`
    Amount        float64 `json:"amount"`
    Processor     string `json:"processor"`
    Status        PaymentStatus `json:"status"`
    StartedAt     time.Time `json:"startedAt"`
}

type PaymentParams struct {
    CorrelationID string `json:"correlationId"`
    Amount        float64 `json:"amount"`
}

type totalPayments struct {
    TotalRequests int `json:"totalRequests"`
    TotalAmount float64 `json:"totalAmount"`
}

type PaymentSummaryParams struct {
    Filter bool `json:"filter"`
    From time.Time `json:"from"`
    To time.Time `json:"to"`
}

type PaymentSummary struct {
    Default totalPayments `json:"default"`
    Fallback totalPayments `json:"fallback"`
}
