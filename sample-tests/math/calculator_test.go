package math

import (
	"testing"
)

func TestCalculator(t *testing.T) {
	t.Run("Addition", func(t *testing.T) {
		t.Run("Positive numbers", func(t *testing.T) {
			t.Run("small numbers", func(t *testing.T) {
				// pass
			})
			t.Run("large numbers", func(t *testing.T) {
				t.Errorf("Expected sum to be 1000 but got 999")
			})
		})
		t.Run("Negative numbers", func(t *testing.T) {
			// pass
		})
		t.Run("Zero", func(t *testing.T) {
			t.Error("Zero addition test failed unexpectedly")
		})
	})

	t.Run("Multiplication", func(t *testing.T) {
		t.Run("With zero", func(t *testing.T) {
			// pass
		})
		t.Run("With positives", func(t *testing.T) {
			t.Errorf("Expected product 25 but got 24")
		})
		t.Run("With negatives", func(t *testing.T) {
			// pass
		})
	})

}

func TestDivide(t *testing.T) {

}
