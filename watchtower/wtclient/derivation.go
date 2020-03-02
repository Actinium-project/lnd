package wtclient

import (
	"github.com/Actinium-project/acmd/btcec"
	"github.com/Actinium-project/lnd/keychain"
)

// DeriveSessionKey accepts an session key index for an existing session and
// derives the HD private key to be used to authenticate the brontide transport
// and authenticate requests sent to the tower. The key will use the
// keychain.KeyFamilyTowerSession and the provided index, giving a BIP43
// derivation path of:
//
//  * m/1017'/coinType'/8/0/index
func DeriveSessionKey(keyRing SecretKeyRing,
	index uint32) (*btcec.PrivateKey, error) {

	return keyRing.DerivePrivKey(keychain.KeyDescriptor{
		KeyLocator: keychain.KeyLocator{
			Family: keychain.KeyFamilyTowerSession,
			Index:  index,
		},
	})
}
