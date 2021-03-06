package lnwallet

import (
	"github.com/Actinium-project/acmutil"
	"github.com/Actinium-project/acmwallet/wallet/txrules"
	"github.com/Actinium-project/lnd/input"
)

// DefaultDustLimit is used to calculate the dust HTLC amount which will be
// send to other node during funding process.
func DefaultDustLimit() acmutil.Amount {
	return txrules.GetDustThreshold(input.P2WSHSize, txrules.DefaultRelayFeePerKb)
}
