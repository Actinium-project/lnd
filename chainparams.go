package lnd

import (
	"github.com/Actinium-project/acmd/chaincfg"
	actiniumCfg "github.com/Actinium-project/acmd/chaincfg"
	bitcoinCfg "github.com/Actinium-project/acmd/chaincfg"
	"github.com/Actinium-project/acmd/chaincfg/chainhash"
	bitcoinWire "github.com/Actinium-project/acmd/wire"
	"github.com/Actinium-project/lnd/keychain"
)

// activeNetParams is a pointer to the parameters specific to the currently
// active bitcoin network.
var activeNetParams = actiniumMainNetParams

// bitcoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type bitcoinNetParams struct {
	*bitcoinCfg.Params
	rpcPort  string
	CoinType uint32
}

// actiniumNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type actiniumNetParams struct {
	*actiniumCfg.Params
	rpcPort  string
	CoinType uint32
}

// bitcoinTestNetParams contains parameters specific to the 3rd version of the
// test network.
var bitcoinTestNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.TestNet4Params,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// bitcoinMainNetParams contains parameters specific to the current Bitcoin
// mainnet.
var bitcoinMainNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.MainNetParams,
	rpcPort:  "8334",
	CoinType: keychain.CoinTypeBitcoin,
}

// bitcoinSimNetParams contains parameters specific to the simulation test
// network.
var bitcoinSimNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.SimNetParams,
	rpcPort:  "18556",
	CoinType: keychain.CoinTypeTestnet,
}

var actiniumSimNetParams = actiniumNetParams{
	Params:   &actiniumCfg.SimNetParams,
	rpcPort:  "18433",
	CoinType: keychain.CoinTypeTestnet,
}

// litecoinTestNetParams contains parameters specific to the 4th version of the
// test network.
var actiniumTestNetParams = actiniumNetParams{
	Params:   &actiniumCfg.TestNet4Params,
	rpcPort:  "19334",
	CoinType: keychain.CoinTypeTestnet,
}

// actiniumMainNetParams contains the parameters specific to the current
// Actinium mainnet.
var actiniumMainNetParams = actiniumNetParams{
	Params:   &actiniumCfg.MainNetParams,
	rpcPort:  "2300",
	CoinType: keychain.CoinTypeActinium,
}

// bitcoinRegTestNetParams contains parameters specific to a local bitcoin
// regtest network.
var bitcoinRegTestNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.RegressionNetParams,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

var actiniumRegTestNetParams = actiniumNetParams{
	Params:   &actiniumCfg.RegressionNetParams,
	rpcPort:  "19335",
	CoinType: keychain.CoinTypeTestnet,
}

// applyActiniumParams applies the relevant chain configuration parameters that
// differ for actinium to the chain parameters typed for btcsuite derivation.
// This function is used in place of using something like interface{} to
// abstract over _which_ chain (or fork) the parameters are for.
func applyActiniumParams(params *actiniumNetParams, actiniumParams *actiniumNetParams) {
	params.Name = actiniumParams.Name
	params.Net = bitcoinWire.BitcoinNet(actiniumParams.Net)
	params.DefaultPort = actiniumParams.DefaultPort
	params.CoinbaseMaturity = actiniumParams.CoinbaseMaturity

	copy(params.GenesisHash[:], actiniumParams.GenesisHash[:])

	// Address encoding magics
	params.PubKeyHashAddrID = actiniumParams.PubKeyHashAddrID
	params.ScriptHashAddrID = actiniumParams.ScriptHashAddrID
	params.PrivateKeyID = actiniumParams.PrivateKeyID
	params.WitnessPubKeyHashAddrID = actiniumParams.WitnessPubKeyHashAddrID
	params.WitnessScriptHashAddrID = actiniumParams.WitnessScriptHashAddrID
	params.Bech32HRPSegwit = actiniumParams.Bech32HRPSegwit

	copy(params.HDPrivateKeyID[:], actiniumParams.HDPrivateKeyID[:])
	copy(params.HDPublicKeyID[:], actiniumParams.HDPublicKeyID[:])

	params.HDCoinType = actiniumParams.HDCoinType

	checkPoints := make([]chaincfg.Checkpoint, len(actiniumParams.Checkpoints))
	for i := 0; i < len(actiniumParams.Checkpoints); i++ {
		var chainHash chainhash.Hash
		copy(chainHash[:], actiniumParams.Checkpoints[i].Hash[:])

		checkPoints[i] = chaincfg.Checkpoint{
			Height: actiniumParams.Checkpoints[i].Height,
			Hash:   &chainHash,
		}
	}
	params.Checkpoints = checkPoints

	params.rpcPort = actiniumParams.rpcPort
	params.CoinType = actiniumParams.CoinType
}

// isTestnet tests if the given params correspond to a testnet
// parameter configuration.
func isTestnet(params *actiniumNetParams) bool {
	switch params.Params.Net {
	case bitcoinWire.TestNet4:
		return true
	default:
		return false
	}
}
