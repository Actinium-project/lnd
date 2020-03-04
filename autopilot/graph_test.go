package autopilot_test

import (
	"testing"

	"github.com/Actinium-project/acmutil"
	"github.com/Actinium-project/lnd/autopilot"
)

// TestMedian tests the Median method.
func TestMedian(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		values []acmutil.Amount
		median acmutil.Amount
	}{
		{
			values: []acmutil.Amount{},
			median: 0,
		},
		{
			values: []acmutil.Amount{10},
			median: 10,
		},
		{
			values: []acmutil.Amount{10, 20},
			median: 15,
		},
		{
			values: []acmutil.Amount{10, 20, 30},
			median: 20,
		},
		{
			values: []acmutil.Amount{30, 10, 20},
			median: 20,
		},
		{
			values: []acmutil.Amount{10, 10, 10, 10, 5000000},
			median: 10,
		},
	}

	for _, test := range testCases {
		res := autopilot.Median(test.values)
		if res != test.median {
			t.Fatalf("expected median %v, got %v", test.median, res)
		}
	}
}
