package core

import (
	"testing"
)

func TestErrorStrategyString(t *testing.T) {
	tests := []struct {
		strategy ErrorStrategy
		expected string
	}{
		{FailFast, "FailFast"},
		{CollectAll, "CollectAll"},
		{ErrorStrategy(999), "Unknown"},
	}

	for _, test := range tests {
		if got := test.strategy.String(); got != test.expected {
			t.Errorf("ErrorStrategy(%d).String() = %q, want %q", test.strategy, got, test.expected)
		}
	}
}
