// Package bmi computes body mass index from weight and height.
package bmi

// Calculate returns the body mass index for a person weighing weightKg
// kilograms with a height of heightCm centimeters.
func Calculate(weightKg, heightCm float64) float64 {
	heightM := heightCm / 100
	return weightKg / (heightM * heightM)
}
