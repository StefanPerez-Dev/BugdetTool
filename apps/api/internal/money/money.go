package money

import (
	"math"
	"strconv"
)

func ToCents(amount float64) int {
	return int(math.Round(amount * 100))
}

func FromCents(cents int) float64 {
	return float64(cents) / 100
}

func CategoryColor(hue *int) string {
	if hue == nil {
		return "oklch(0.68 0 0)"
	}
	return "oklch(0.64 0.15 " + strconv.Itoa(*hue) + ")"
}
