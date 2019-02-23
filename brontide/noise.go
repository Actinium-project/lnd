package brontide

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"

	"github.com/Actinium-project/acmd/btcec"
	"github.com/lightningnetwork/lnd/buffer"
	"github.com/lightningnetwork/lnd/pool"
)

const (
	// protocolName is the precise instantiation of the Noise protocol
	// handshake at the center of Brontide. This value will be used as part
	// of the prologue. If the initiator and responder aren't using the
	// exact same string for this value, along with prologue of the Bitcoin
	// network, then the initial handshake will fail.
	protocolName = "Noise_XK_secp256k1_ChaChaPoly_SHA256"

	// macSize is the length in bytes of the tags generated by poly1305.
	macSize = 16

	// lengthHeaderSize is the number of bytes used to prefix encode the
	// length of a message payload.
	lengthHeaderSize = 2

	// keyRotationInterval is the number of messages sent on a single
	// cipher stream before the keys are rotated forwards.
	keyRotationInterval = 1000

	// handshakeReadTimeout is a read timeout that will be enforced when
	// waiting for data payloads during the various acts of Brontide. If
	// the remote party fails to deliver the proper payload within this
	// time frame, then we'll fail the connection.
	handshakeReadTimeout = time.Second * 5
)

var (
	// ErrMaxMessageLengthExceeded is returned a message to be written to
	// the cipher session exceeds the maximum allowed message payload.
	ErrMaxMessageLengthExceeded = errors.New("the generated payload exceeds " +
		"the max allowed message length of (2^16)-1")

	// lightningPrologue is the noise prologue that is used to initialize
	// the brontide noise handshake.
	lightningPrologue = []byte("lightning")

	// ephemeralGen is the default ephemeral key generator, used to derive a
	// unique ephemeral key for each brontide handshake.
	ephemeralGen = func() (*btcec.PrivateKey, error) {
		return btcec.NewPrivateKey(btcec.S256())
	}

	// readBufferPool is a singleton instance of a buffer pool, used to
	// conserve memory allocations due to read buffers across the entire
	// brontide package.
	readBufferPool = pool.NewReadBuffer(
		pool.DefaultReadBufferGCInterval,
		pool.DefaultReadBufferExpiryInterval,
	)
)

// TODO(roasbeef): free buffer pool?

// ecdh performs an ECDH operation between pub and priv. The returned value is
// the sha256 of the compressed shared point.
func ecdh(pub *btcec.PublicKey, priv *btcec.PrivateKey) []byte {
	s := &btcec.PublicKey{}
	x, y := btcec.S256().ScalarMult(pub.X, pub.Y, priv.D.Bytes())
	s.X = x
	s.Y = y

	h := sha256.Sum256(s.SerializeCompressed())
	return h[:]
}

// cipherState encapsulates the state for the AEAD which will be used to
// encrypt+authenticate any payloads sent during the handshake, and messages
// sent once the handshake has completed.
type cipherState struct {
	// nonce is the nonce passed into the chacha20-poly1305 instance for
	// encryption+decryption. The nonce is incremented after each successful
	// encryption/decryption.
	//
	// TODO(roasbeef): this should actually be 96 bit
	nonce uint64

	// secretKey is the shared symmetric key which will be used to
	// instantiate the cipher.
	//
	// TODO(roasbeef): m-lock??
	secretKey [32]byte

	// salt is an additional secret which is used during key rotation to
	// generate new keys.
	salt [32]byte

	// cipher is an instance of the ChaCha20-Poly1305 AEAD construction
	// created using the secretKey above.
	cipher cipher.AEAD
}

// Encrypt returns a ciphertext which is the encryption of the plainText
// observing the passed associatedData within the AEAD construction.
func (c *cipherState) Encrypt(associatedData, cipherText, plainText []byte) []byte {
	defer func() {
		c.nonce++

		if c.nonce == keyRotationInterval {
			c.rotateKey()
		}
	}()

	var nonce [12]byte
	binary.LittleEndian.PutUint64(nonce[4:], c.nonce)

	return c.cipher.Seal(cipherText, nonce[:], plainText, associatedData)
}

// Decrypt attempts to decrypt the passed ciphertext observing the specified
// associatedData within the AEAD construction. In the case that the final MAC
// check fails, then a non-nil error will be returned.
func (c *cipherState) Decrypt(associatedData, plainText, cipherText []byte) ([]byte, error) {
	defer func() {
		c.nonce++

		if c.nonce == keyRotationInterval {
			c.rotateKey()
		}
	}()

	var nonce [12]byte
	binary.LittleEndian.PutUint64(nonce[4:], c.nonce)

	return c.cipher.Open(plainText, nonce[:], cipherText, associatedData)
}

// InitializeKey initializes the secret key and AEAD cipher scheme based off of
// the passed key.
func (c *cipherState) InitializeKey(key [32]byte) {
	c.secretKey = key
	c.nonce = 0

	// Safe to ignore the error here as our key is properly sized
	// (32-bytes).
	c.cipher, _ = chacha20poly1305.New(c.secretKey[:])
}

// InitializeKeyWithSalt is identical to InitializeKey however it also sets the
// cipherState's salt field which is used for key rotation.
func (c *cipherState) InitializeKeyWithSalt(salt, key [32]byte) {
	c.salt = salt
	c.InitializeKey(key)
}

// rotateKey rotates the current encryption/decryption key for this cipherState
// instance. Key rotation is performed by ratcheting the current key forward
// using an HKDF invocation with the cipherState's salt as the salt, and the
// current key as the input.
func (c *cipherState) rotateKey() {
	var (
		info    []byte
		nextKey [32]byte
	)

	oldKey := c.secretKey
	h := hkdf.New(sha256.New, oldKey[:], c.salt[:], info)

	// hkdf(ck, k, zero)
	// |
	// | \
	// |  \
	// ck  k'
	h.Read(c.salt[:])
	h.Read(nextKey[:])

	c.InitializeKey(nextKey)
}

// symmetricState encapsulates a cipherState object and houses the ephemeral
// handshake digest state. This struct is used during the handshake to derive
// new shared secrets based off of the result of ECDH operations. Ultimately,
// the final key yielded by this struct is the result of an incremental
// Triple-DH operation.
type symmetricState struct {
	cipherState

	// chainingKey is used as the salt to the HKDF function to derive a new
	// chaining key as well as a new tempKey which is used for
	// encryption/decryption.
	chainingKey [32]byte

	// tempKey is the latter 32 bytes resulted from the latest HKDF
	// iteration. This key is used to encrypt/decrypt any handshake
	// messages or payloads sent until the next DH operation is executed.
	tempKey [32]byte

	// handshakeDigest is the cumulative hash digest of all handshake
	// messages sent from start to finish. This value is never transmitted
	// to the other side, but will be used as the AD when
	// encrypting/decrypting messages using our AEAD construction.
	handshakeDigest [32]byte
}

// mixKey is implements a basic HKDF-based key ratchet. This method is called
// with the result of each DH output generated during the handshake process.
// The first 32 bytes extract from the HKDF reader is the next chaining key,
// then latter 32 bytes become the temp secret key using within any future AEAD
// operations until another DH operation is performed.
func (s *symmetricState) mixKey(input []byte) {
	var info []byte

	secret := input
	salt := s.chainingKey
	h := hkdf.New(sha256.New, secret, salt[:], info)

	// hkdf(ck, input, zero)
	// |
	// | \
	// |  \
	// ck  k
	h.Read(s.chainingKey[:])
	h.Read(s.tempKey[:])

	// cipher.k = temp_key
	s.InitializeKey(s.tempKey)
}

// mixHash hashes the passed input data into the cumulative handshake digest.
// The running result of this value (h) is used as the associated data in all
// decryption/encryption operations.
func (s *symmetricState) mixHash(data []byte) {
	h := sha256.New()
	h.Write(s.handshakeDigest[:])
	h.Write(data)

	copy(s.handshakeDigest[:], h.Sum(nil))
}

// EncryptAndHash returns the authenticated encryption of the passed plaintext.
// When encrypting the handshake digest (h) is used as the associated data to
// the AEAD cipher.
func (s *symmetricState) EncryptAndHash(plaintext []byte) []byte {
	ciphertext := s.Encrypt(s.handshakeDigest[:], nil, plaintext)

	s.mixHash(ciphertext)

	return ciphertext
}

// DecryptAndHash returns the authenticated decryption of the passed
// ciphertext.  When encrypting the handshake digest (h) is used as the
// associated data to the AEAD cipher.
func (s *symmetricState) DecryptAndHash(ciphertext []byte) ([]byte, error) {
	plaintext, err := s.Decrypt(s.handshakeDigest[:], nil, ciphertext)
	if err != nil {
		return nil, err
	}

	s.mixHash(ciphertext)

	return plaintext, nil
}

// InitializeSymmetric initializes the symmetric state by setting the handshake
// digest (h) and the chaining key (ck) to protocol name.
func (s *symmetricState) InitializeSymmetric(protocolName []byte) {
	var empty [32]byte

	s.handshakeDigest = sha256.Sum256(protocolName)
	s.chainingKey = s.handshakeDigest
	s.InitializeKey(empty)
}

// handshakeState encapsulates the symmetricState and keeps track of all the
// public keys (static and ephemeral) for both sides during the handshake
// transcript. If the handshake completes successfully, then two instances of a
// cipherState are emitted: one to encrypt messages from initiator to
// responder, and the other for the opposite direction.
type handshakeState struct {
	symmetricState

	initiator bool

	localStatic    *btcec.PrivateKey
	localEphemeral *btcec.PrivateKey

	remoteStatic    *btcec.PublicKey
	remoteEphemeral *btcec.PublicKey
}

// newHandshakeState returns a new instance of the handshake state initialized
// with the prologue and protocol name. If this is the responder's handshake
// state, then the remotePub can be nil.
func newHandshakeState(initiator bool, prologue []byte,
	localPub *btcec.PrivateKey, remotePub *btcec.PublicKey) handshakeState {

	h := handshakeState{
		initiator:    initiator,
		localStatic:  localPub,
		remoteStatic: remotePub,
	}

	// Set the current chaining key and handshake digest to the hash of the
	// protocol name, and additionally mix in the prologue. If either sides
	// disagree about the prologue or protocol name, then the handshake
	// will fail.
	h.InitializeSymmetric([]byte(protocolName))
	h.mixHash(prologue)

	// In Noise_XK, then initiator should know the responder's static
	// public key, therefore we include the responder's static key in the
	// handshake digest. If the initiator gets this value wrong, then the
	// handshake will fail.
	if initiator {
		h.mixHash(remotePub.SerializeCompressed())
	} else {
		h.mixHash(localPub.PubKey().SerializeCompressed())
	}

	return h
}

// EphemeralGenerator is a functional option that allows callers to substitute
// a custom function for use when generating ephemeral keys for ActOne or
// ActTwo.  The function closure return by this function can be passed into
// NewBrontideMachine as a function option parameter.
func EphemeralGenerator(gen func() (*btcec.PrivateKey, error)) func(*Machine) {
	return func(m *Machine) {
		m.ephemeralGen = gen
	}
}

// Machine is a state-machine which implements Brontide: an
// Authenticated-key Exchange in Three Acts. Brontide is derived from the Noise
// framework, specifically implementing the Noise_XK handshake. Once the
// initial 3-act handshake has completed all messages are encrypted with a
// chacha20 AEAD cipher. On the wire, all messages are prefixed with an
// authenticated+encrypted length field. Additionally, the encrypted+auth'd
// length prefix is used as the AD when encrypting+decryption messages. This
// construction provides confidentiality of packet length, avoids introducing
// a padding-oracle, and binds the encrypted packet length to the packet
// itself.
//
// The acts proceeds the following order (initiator on the left):
//  GenActOne()   ->
//                    RecvActOne()
//                <-  GenActTwo()
//  RecvActTwo()
//  GenActThree() ->
//                    RecvActThree()
//
// This exchange corresponds to the following Noise handshake:
//   <- s
//   ...
//   -> e, es
//   <- e, ee
//   -> s, se
type Machine struct {
	sendCipher cipherState
	recvCipher cipherState

	ephemeralGen func() (*btcec.PrivateKey, error)

	handshakeState

	// nextCipherHeader is a static buffer that we'll use to read in the
	// next ciphertext header from the wire. The header is a 2 byte length
	// (of the next ciphertext), followed by a 16 byte MAC.
	nextCipherHeader [lengthHeaderSize + macSize]byte

	// nextCipherText is a static buffer that we'll use to read in the
	// bytes of the next cipher text message. As all messages in the
	// protocol MUST be below 65KB plus our macSize, this will be
	// sufficient to buffer all messages from the socket when we need to
	// read the next one. Having a fixed buffer that's re-used also means
	// that we save on allocations as we don't need to create a new one
	// each time.
	nextCipherText *buffer.Read
}

// NewBrontideMachine creates a new instance of the brontide state-machine. If
// the responder (listener) is creating the object, then the remotePub should
// be nil. The handshake state within brontide is initialized using the ascii
// string "lightning" as the prologue. The last parameter is a set of variadic
// arguments for adding additional options to the brontide Machine
// initialization.
func NewBrontideMachine(initiator bool, localPub *btcec.PrivateKey,
	remotePub *btcec.PublicKey, options ...func(*Machine)) *Machine {

	handshake := newHandshakeState(
		initiator, lightningPrologue, localPub, remotePub,
	)

	m := &Machine{
		handshakeState: handshake,
		ephemeralGen:   ephemeralGen,
	}

	// With the default options established, we'll now process all the
	// options passed in as parameters.
	for _, option := range options {
		option(m)
	}

	return m
}

const (
	// HandshakeVersion is the expected version of the brontide handshake.
	// Any messages that carry a different version will cause the handshake
	// to abort immediately.
	HandshakeVersion = byte(0)

	// ActOneSize is the size of the packet sent from initiator to
	// responder in ActOne. The packet consists of a handshake version, an
	// ephemeral key in compressed format, and a 16-byte poly1305 tag.
	//
	// 1 + 33 + 16
	ActOneSize = 50

	// ActTwoSize is the size the packet sent from responder to initiator
	// in ActTwo. The packet consists of a handshake version, an ephemeral
	// key in compressed format and a 16-byte poly1305 tag.
	//
	// 1 + 33 + 16
	ActTwoSize = 50

	// ActThreeSize is the size of the packet sent from initiator to
	// responder in ActThree. The packet consists of a handshake version,
	// the initiators static key encrypted with strong forward secrecy and
	// a 16-byte poly1035
	// tag.
	//
	// 1 + 33 + 16 + 16
	ActThreeSize = 66
)

// GenActOne generates the initial packet (act one) to be sent from initiator
// to responder. During act one the initiator generates a fresh ephemeral key,
// hashes it into the handshake digest, and performs an ECDH between this key
// and the responder's static key. Future payloads are encrypted with a key
// derived from this result.
//
//    -> e, es
func (b *Machine) GenActOne() ([ActOneSize]byte, error) {
	var (
		err    error
		actOne [ActOneSize]byte
	)

	// e
	b.localEphemeral, err = b.ephemeralGen()
	if err != nil {
		return actOne, err
	}

	ephemeral := b.localEphemeral.PubKey().SerializeCompressed()
	b.mixHash(ephemeral)

	// es
	s := ecdh(b.remoteStatic, b.localEphemeral)
	b.mixKey(s[:])

	authPayload := b.EncryptAndHash([]byte{})

	actOne[0] = HandshakeVersion
	copy(actOne[1:34], ephemeral)
	copy(actOne[34:], authPayload)

	return actOne, nil
}

// RecvActOne processes the act one packet sent by the initiator. The responder
// executes the mirrored actions to that of the initiator extending the
// handshake digest and deriving a new shared secret based on an ECDH with the
// initiator's ephemeral key and responder's static key.
func (b *Machine) RecvActOne(actOne [ActOneSize]byte) error {
	var (
		err error
		e   [33]byte
		p   [16]byte
	)

	// If the handshake version is unknown, then the handshake fails
	// immediately.
	if actOne[0] != HandshakeVersion {
		return fmt.Errorf("Act One: invalid handshake version: %v, "+
			"only %v is valid, msg=%x", actOne[0], HandshakeVersion,
			actOne[:])
	}

	copy(e[:], actOne[1:34])
	copy(p[:], actOne[34:])

	// e
	b.remoteEphemeral, err = btcec.ParsePubKey(e[:], btcec.S256())
	if err != nil {
		return err
	}
	b.mixHash(b.remoteEphemeral.SerializeCompressed())

	// es
	s := ecdh(b.remoteEphemeral, b.localStatic)
	b.mixKey(s)

	// If the initiator doesn't know our static key, then this operation
	// will fail.
	_, err = b.DecryptAndHash(p[:])
	return err
}

// GenActTwo generates the second packet (act two) to be sent from the
// responder to the initiator. The packet for act two is identify to that of
// act one, but then results in a different ECDH operation between the
// initiator's and responder's ephemeral keys.
//
//    <- e, ee
func (b *Machine) GenActTwo() ([ActTwoSize]byte, error) {
	var (
		err    error
		actTwo [ActTwoSize]byte
	)

	// e
	b.localEphemeral, err = b.ephemeralGen()
	if err != nil {
		return actTwo, err
	}

	ephemeral := b.localEphemeral.PubKey().SerializeCompressed()
	b.mixHash(b.localEphemeral.PubKey().SerializeCompressed())

	// ee
	s := ecdh(b.remoteEphemeral, b.localEphemeral)
	b.mixKey(s)

	authPayload := b.EncryptAndHash([]byte{})

	actTwo[0] = HandshakeVersion
	copy(actTwo[1:34], ephemeral)
	copy(actTwo[34:], authPayload)

	return actTwo, nil
}

// RecvActTwo processes the second packet (act two) sent from the responder to
// the initiator. A successful processing of this packet authenticates the
// initiator to the responder.
func (b *Machine) RecvActTwo(actTwo [ActTwoSize]byte) error {
	var (
		err error
		e   [33]byte
		p   [16]byte
	)

	// If the handshake version is unknown, then the handshake fails
	// immediately.
	if actTwo[0] != HandshakeVersion {
		return fmt.Errorf("Act Two: invalid handshake version: %v, "+
			"only %v is valid, msg=%x", actTwo[0], HandshakeVersion,
			actTwo[:])
	}

	copy(e[:], actTwo[1:34])
	copy(p[:], actTwo[34:])

	// e
	b.remoteEphemeral, err = btcec.ParsePubKey(e[:], btcec.S256())
	if err != nil {
		return err
	}
	b.mixHash(b.remoteEphemeral.SerializeCompressed())

	// ee
	s := ecdh(b.remoteEphemeral, b.localEphemeral)
	b.mixKey(s)

	_, err = b.DecryptAndHash(p[:])
	return err
}

// GenActThree creates the final (act three) packet of the handshake. Act three
// is to be sent from the initiator to the responder. The purpose of act three
// is to transmit the initiator's public key under strong forward secrecy to
// the responder. This act also includes the final ECDH operation which yields
// the final session.
//
//    -> s, se
func (b *Machine) GenActThree() ([ActThreeSize]byte, error) {
	var actThree [ActThreeSize]byte

	ourPubkey := b.localStatic.PubKey().SerializeCompressed()
	ciphertext := b.EncryptAndHash(ourPubkey)

	s := ecdh(b.remoteEphemeral, b.localStatic)
	b.mixKey(s)

	authPayload := b.EncryptAndHash([]byte{})

	actThree[0] = HandshakeVersion
	copy(actThree[1:50], ciphertext)
	copy(actThree[50:], authPayload)

	// With the final ECDH operation complete, derive the session sending
	// and receiving keys.
	b.split()

	return actThree, nil
}

// RecvActThree processes the final act (act three) sent from the initiator to
// the responder. After processing this act, the responder learns of the
// initiator's static public key. Decryption of the static key serves to
// authenticate the initiator to the responder.
func (b *Machine) RecvActThree(actThree [ActThreeSize]byte) error {
	var (
		err error
		s   [33 + 16]byte
		p   [16]byte
	)

	// If the handshake version is unknown, then the handshake fails
	// immediately.
	if actThree[0] != HandshakeVersion {
		return fmt.Errorf("Act Three: invalid handshake version: %v, "+
			"only %v is valid, msg=%x", actThree[0], HandshakeVersion,
			actThree[:])
	}

	copy(s[:], actThree[1:33+16+1])
	copy(p[:], actThree[33+16+1:])

	// s
	remotePub, err := b.DecryptAndHash(s[:])
	if err != nil {
		return err
	}
	b.remoteStatic, err = btcec.ParsePubKey(remotePub, btcec.S256())
	if err != nil {
		return err
	}

	// se
	se := ecdh(b.remoteStatic, b.localEphemeral)
	b.mixKey(se)

	if _, err := b.DecryptAndHash(p[:]); err != nil {
		return err
	}

	// With the final ECDH operation complete, derive the session sending
	// and receiving keys.
	b.split()

	return nil
}

// split is the final wrap-up act to be executed at the end of a successful
// three act handshake. This function creates two internal cipherState
// instances: one which is used to encrypt messages from the initiator to the
// responder, and another which is used to encrypt message for the opposite
// direction.
func (b *Machine) split() {
	var (
		empty   []byte
		sendKey [32]byte
		recvKey [32]byte
	)

	h := hkdf.New(sha256.New, empty, b.chainingKey[:], empty)

	// If we're the initiator the first 32 bytes are used to encrypt our
	// messages and the second 32-bytes to decrypt their messages. For the
	// responder the opposite is true.
	if b.initiator {
		h.Read(sendKey[:])
		b.sendCipher = cipherState{}
		b.sendCipher.InitializeKeyWithSalt(b.chainingKey, sendKey)

		h.Read(recvKey[:])
		b.recvCipher = cipherState{}
		b.recvCipher.InitializeKeyWithSalt(b.chainingKey, recvKey)
	} else {
		h.Read(recvKey[:])
		b.recvCipher = cipherState{}
		b.recvCipher.InitializeKeyWithSalt(b.chainingKey, recvKey)

		h.Read(sendKey[:])
		b.sendCipher = cipherState{}
		b.sendCipher.InitializeKeyWithSalt(b.chainingKey, sendKey)
	}
}

// WriteMessage writes the next message p to the passed io.Writer. The
// ciphertext of the message is prepended with an encrypt+auth'd length which
// must be used as the AD to the AEAD construction when being decrypted by the
// other side.
func (b *Machine) WriteMessage(w io.Writer, p []byte) error {
	// The total length of each message payload including the MAC size
	// payload exceed the largest number encodable within a 16-bit unsigned
	// integer.
	if len(p) > math.MaxUint16 {
		return ErrMaxMessageLengthExceeded
	}

	// The full length of the packet is only the packet length, and does
	// NOT include the MAC.
	fullLength := uint16(len(p))

	var pktLen [2]byte
	binary.BigEndian.PutUint16(pktLen[:], fullLength)

	// First, write out the encrypted+MAC'd length prefix for the packet.
	cipherLen := b.sendCipher.Encrypt(nil, nil, pktLen[:])
	if _, err := w.Write(cipherLen); err != nil {
		return err
	}

	// Finally, write out the encrypted packet itself. We only write out a
	// single packet, as any fragmentation should have taken place at a
	// higher level.
	cipherText := b.sendCipher.Encrypt(nil, nil, p)
	_, err := w.Write(cipherText)
	return err
}

// ReadMessage attempts to read the next message from the passed io.Reader. In
// the case of an authentication error, a non-nil error is returned.
func (b *Machine) ReadMessage(r io.Reader) ([]byte, error) {
	if _, err := io.ReadFull(r, b.nextCipherHeader[:]); err != nil {
		return nil, err
	}

	// Attempt to decrypt+auth the packet length present in the stream.
	pktLenBytes, err := b.recvCipher.Decrypt(
		nil, nil, b.nextCipherHeader[:],
	)
	if err != nil {
		return nil, err
	}

	// If this is the first message being read, take a read buffer from the
	// buffer pool. This is delayed until this point to avoid allocating
	// read buffers until after the peer has successfully completed the
	// handshake, and is ready to begin sending lnwire messages.
	if b.nextCipherText == nil {
		b.nextCipherText = readBufferPool.Take()
		runtime.SetFinalizer(b, freeReadBuffer)
	}

	// Next, using the length read from the packet header, read the
	// encrypted packet itself.
	pktLen := uint32(binary.BigEndian.Uint16(pktLenBytes)) + macSize
	if _, err := io.ReadFull(r, b.nextCipherText[:pktLen]); err != nil {
		return nil, err
	}

	// TODO(roasbeef): modify to let pass in slice
	return b.recvCipher.Decrypt(nil, nil, b.nextCipherText[:pktLen])
}

// freeReadBuffer returns the Machine's read buffer back to the package wide
// read buffer pool.
//
// NOTE: This method should only be called by a Machine's finalizer.
func freeReadBuffer(b *Machine) {
	readBufferPool.Return(b.nextCipherText)
	b.nextCipherText = nil
}
