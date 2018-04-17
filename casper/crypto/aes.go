package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"

	pbkdf2 "golang.org/x/crypto/pbkdf2"
)

var log = logging.Logger("crypto")

const (
	AESKeySize = 32
	SaltSize   = 16
	iterCount  = 10000
)

func genSalt() ([]byte, error) {
	var salt = make([]byte, SaltSize)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return nil, err
	}

	return salt, nil
}

func getKeyNonce(passwd []byte, salt []byte) (key []byte, nonce []byte) {
	tmpbuf := pbkdf2.Key(passwd, salt, iterCount, AESKeySize+aes.BlockSize, sha256.New)
	return tmpbuf[:AESKeySize], tmpbuf[AESKeySize:]
}

func NewAESReaderWithSalt(r io.Reader, passwd []byte, salt []byte) io.Reader {
	key, nonce := getKeyNonce(passwd, salt)
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	ctr := cipher.NewCTR(c, nonce)
	return cipher.StreamReader{
		S: ctr,
		R: r,
	}
}

func NewAESReader(r io.Reader, passwd []byte) io.Reader {
	var salt = make([]byte, SaltSize)
	_, err := io.ReadFull(r, salt)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	key, nonce := getKeyNonce(passwd, salt)
	c, err := aes.NewCipher(key)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	ctr := cipher.NewCTR(c, nonce)
	return cipher.StreamReader{
		S: ctr,
		R: r,
	}
}

type aesReadCloser struct {
	SR io.Reader
	r  *io.ReadCloser
}

func (arc aesReadCloser) Read(p []byte) (int, error) {
	return arc.SR.Read(p)
}

func (arc aesReadCloser) Close() error {
	return (*arc.r).Close()
}

func NewAESEncryptReadCloser(r io.ReadCloser, passwd []byte) io.ReadCloser {
	var salt, err = genSalt()

	key, nonce := getKeyNonce(passwd, salt)
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	ctr := cipher.NewCTR(c, nonce)
	return &aesReadCloser{
		SR: io.MultiReader(bytes.NewReader(salt), cipher.StreamReader{
			S: ctr,
			R: r,
		}),
		r: &r,
	}
}
