package autopilot

import (
	"testing"
	"time"

	prand "math/rand"

	"github.com/Actinium-project/lnd/lnwire"
	"github.com/Actinium-project/acmutil"
)

func TestConstraintsChannelBudget(t *testing.T) {
	t.Parallel()

	prand.Seed(time.Now().Unix())

	const (
		minChanSize = 0
		maxChanSize = acmutil.Amount(acmutil.SatoshiPerBitcoin)

		chanLimit = 3

		threshold = 0.5
	)

	constraints := NewConstraints(
		minChanSize,
		maxChanSize,
		chanLimit,
		0,
		threshold,
	)

	randChanID := func() lnwire.ShortChannelID {
		return lnwire.NewShortChanIDFromInt(uint64(prand.Int63()))
	}

	testCases := []struct {
		channels  []Channel
		walletAmt acmutil.Amount

		needMore     bool
		amtAvailable acmutil.Amount
		numMore      uint32
	}{
		// Many available funds, but already have too many active open
		// channels.
		{
			[]Channel{
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(prand.Int31()),
				},
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(prand.Int31()),
				},
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(prand.Int31()),
				},
			},
			acmutil.Amount(acmutil.SatoshiPerBitcoin * 10),
			false,
			0,
			0,
		},

		// Ratio of funds in channels and total funds meets the
		// threshold.
		{
			[]Channel{
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(acmutil.SatoshiPerBitcoin),
				},
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(acmutil.SatoshiPerBitcoin),
				},
			},
			acmutil.Amount(acmutil.SatoshiPerBitcoin * 2),
			false,
			0,
			0,
		},

		// Ratio of funds in channels and total funds is below the
		// threshold. We have 10 ACM allocated amongst channels and
		// funds, atm. We're targeting 50%, so 5 ACM should be
		// allocated. Only 1 ACM is atm, so 4 ACM should be
		// recommended. We should also request 2 more channels as the
		// limit is 3.
		{
			[]Channel{
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(acmutil.SatoshiPerBitcoin),
				},
			},
			acmutil.Amount(acmutil.SatoshiPerBitcoin * 9),
			true,
			acmutil.Amount(acmutil.SatoshiPerBitcoin * 4),
			2,
		},

		// Ratio of funds in channels and total funds is below the
		// threshold. We have 14 ACM total amongst the wallet's
		// balance, and our currently opened channels. Since we're
		// targeting a 50% allocation, we should commit 7 ACM. The
		// current channels commit 4 ACM, so we should expected 3 ACM
		// to be committed. We should only request a single additional
		// channel as the limit is 3.
		{
			[]Channel{
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(acmutil.SatoshiPerBitcoin),
				},
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(acmutil.SatoshiPerBitcoin * 3),
				},
			},
			acmutil.Amount(acmutil.SatoshiPerBitcoin * 10),
			true,
			acmutil.Amount(acmutil.SatoshiPerBitcoin * 3),
			1,
		},

		// Ratio of funds in channels and total funds is above the
		// threshold.
		{
			[]Channel{
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(acmutil.SatoshiPerBitcoin),
				},
				{
					ChanID:   randChanID(),
					Capacity: acmutil.Amount(acmutil.SatoshiPerBitcoin),
				},
			},
			acmutil.Amount(acmutil.SatoshiPerBitcoin),
			false,
			0,
			0,
		},
	}

	for i, testCase := range testCases {
		amtToAllocate, numMore := constraints.ChannelBudget(
			testCase.channels, testCase.walletAmt,
		)

		if amtToAllocate != testCase.amtAvailable {
			t.Fatalf("test #%v: expected %v, got %v",
				i, testCase.amtAvailable, amtToAllocate)
		}
		if numMore != testCase.numMore {
			t.Fatalf("test #%v: expected %v, got %v",
				i, testCase.numMore, numMore)
		}
	}
}
