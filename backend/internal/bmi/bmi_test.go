package bmi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/isAdamBailey/massa/backend/internal/bmi"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name     string
		weightKg float64
		heightCm float64
		want     float64
	}{
		{name: "normal", weightKg: 70, heightCm: 175, want: 22.857142857142858},
		{name: "tall", weightKg: 90, heightCm: 200, want: 22.5},
		{name: "short", weightKg: 50, heightCm: 150, want: 22.222222222222225},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, bmi.Calculate(tt.weightKg, tt.heightCm), 1e-9)
		})
	}
}
