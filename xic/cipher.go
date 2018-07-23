package xic

import (
	"fmt"
	"bytes"
	"crypto/aes"
	crand "crypto/rand"

	"halftwo/mangos/eax"
)

type _CipherSuite int

const (
	UNKNOWN_SUITE _CipherSuite  = iota
	CLEARTEXT
	AES128_EAX
	AES192_EAX
	AES256_EAX
	MAX_SUITE
)

func (c _CipherSuite) String() string {
	switch c {
	case UNKNOWN_SUITE:
		return "UNKNOWN"
	case CLEARTEXT:
		return "CLEARTEXT"
	case AES128_EAX:
		return "AES128_EAX"
	case AES192_EAX:
		return "AES192_EAX"
	case AES256_EAX:
		return "AES256_EAX"
	}
	return "INVALID"
}

/*
   After encypting, the msg is:
		IV[16] + CIPHERTEXT + MAC[16]
   The IV is:
		RANDOM[8] + COUNTER[8]
   The MAC is generated by CMAC algorithm
*/


type _Cipher struct {
	rOff int	// RANDOM offset in nonce
	cOff int	// COUNTER offset in nonce

	// The nonce for EAX is: salt + IV[16]
	// The IV[16] is: RANDOM[8] + COUNTER[8]
	ox *eax.EaxCtx
	oNonce []byte	// encrypt
	oMAC [16]byte

	ix *eax.EaxCtx
	iNonce []byte	// decrypt
}



func newXicCipher(suite _CipherSuite, keyInfo []byte, isServer bool) (*_Cipher, error) {
	keyLen := 0
	switch suite {
	case AES128_EAX:
		keyLen = 16
	case AES192_EAX:
		keyLen = 24
	case AES256_EAX:
		keyLen = 32
	default:
                return nil, fmt.Errorf("Unsupported CipherSuite %s", suite)
	}

	c := &_Cipher{}

	n := len(keyInfo) - keyLen
	if n > 16 {
		n = 16
	} else if n < 0 {
		n = 0
	}

	var key [32]byte
	copy(key[:keyLen], keyInfo)

	c.oNonce = make([]byte, n + 16)
	c.iNonce = make([]byte, n + 16)
	if n > 0 {
		copy(c.oNonce[:n], keyInfo[keyLen:])
		copy(c.iNonce[:n], keyInfo[keyLen:])
	}
	c.rOff = n
	c.cOff = c.rOff + 8

	blockCipher, err := aes.NewCipher(key[:keyLen])
	if err != nil {
		return nil, err
	}

	c.ox, err = eax.NewEax(blockCipher)
	if err != nil {
		panic("Can't reach here")
	}

	c.ix, err = eax.NewEax(blockCipher)
	if err != nil {
		panic("Can't reach here")
	}

	if (isServer) {
		c.oNonce[c.cOff] = 0x80
	} else {
		c.iNonce[c.cOff] = 0x80
	}
	return c, nil
}

func increaseCounter(counter []byte) {
	on := (counter[0] & 0x80) != 0
	for i := 7; i >= 0; i-- {
		counter[i]++
                if counter[i] != 0 {
			break
		}
	}

	if on {
		counter[0] |= 0x80
	} else {
		counter[0] &= 0x7f
	}
}

func getRandomBytes(buf []byte) error {
	_, err := crand.Read(buf)
	return err
}

func (c *_Cipher) OutputGetIV(IV []byte) bool {
	increaseCounter(c.oNonce[c.cOff:])

	err := getRandomBytes(c.oNonce[c.rOff:c.cOff])
	if err != nil {
		return false
	}
	copy(IV, c.oNonce[c.rOff:])
	return true
}

func (c *_Cipher) OutputStart(header []byte) {
	c.ox.Start(true, c.oNonce, header)
}

func (c *_Cipher) OutputUpdate(out, in []byte) {
	c.ox.Update(out, in)
}

func (c *_Cipher) OutputMakeMAC() []byte {
	c.ox.Finish(c.oMAC[:])
	return c.oMAC[:]
}


func (c *_Cipher) InputSetIV(IV []byte) bool {
	increaseCounter(c.iNonce[c.cOff:])

	if !bytes.Equal(c.iNonce[c.cOff:], IV[8:16]) {
		return false
	}

	copy(c.iNonce[c.rOff:], IV[:8])
	return true
}

func (c *_Cipher) InputStart(header []byte) {
	c.ix.Start(false, c.iNonce, header)
}

func (c *_Cipher) InputUpdate(out, in []byte) {
	c.ix.Update(out, in)
}

func (c *_Cipher) InputCheckMAC(MAC []byte) bool {
	var mac [16]byte
	c.ix.Finish(mac[:])
	return bytes.Equal(MAC, mac[:])
}


