package money_test

import (
	"testing"

	"github.com/oprimogus/rinha-backend-2025/internal/core/money"
)

func TestToCents(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		amountExpected int
	}{
		{"should return 10000", 100.00, 10000},
		{"should return 1000", 10.00, 1000},
		{"should return 100", 1.0, 100},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := money.ToCents(tt.amount)
			if result != tt.amountExpected {
				t.Errorf("ToCents(%f) = %d, want %d", tt.amount, result, tt.amountExpected)
			}
		})
	}
}
