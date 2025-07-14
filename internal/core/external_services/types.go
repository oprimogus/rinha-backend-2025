package externalservices

import "time"

type PaymentParams struct {
    CorrelationID string `json:"correlationId"`
    Amount        float64 `json:"amount"`
    RequestedAt     time.Time `json:"requestedAt"`
}

type PaymentResponse struct {
    Message string `json:"message"`
}

type HealthCheckResponse struct {
    Failing bool `json:"failing"`
    MinResponseTime int `json:"minResponseTime"`
}
