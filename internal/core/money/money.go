package money

import (
	"math"
	"strconv"
)

func ToCents(amount float64) int {
    return int(math.Round(amount * 100))
}

func ToFloat(amount int) float64 {
    return float64(amount) / 100
}

func FromStringToCents(amount string) (int, error) {
    cents, err := strconv.Atoi(amount)
    if err != nil {
        return 0, err
    }
    return cents, nil
}

func FromStringToFloat(amount string) (float64, error) {
    cents, err := FromStringToCents(amount)
    if err != nil {
        return 0, err
    }
    return ToFloat(cents), nil
}
