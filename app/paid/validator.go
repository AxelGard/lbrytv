package paid

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
)

type pubKeyManager struct {
	key *rsa.PublicKey
}

var pubKM *pubKeyManager

// InitPubKey should be called with pubkey url as an argument before CanPlayStream can be called
func InitPubKey(rawKey []byte) error {
	k := &pubKeyManager{}
	k.loadFromBytes(rawKey)
	pubKM = k
	return nil
}

// CanPlayStream is the main entry point for players to validate paid media tokens
func CanPlayStream(sdHash string, stringToken string) (bool, error) {
	t, err := pubKM.ValidateToken(stringToken)
	if err != nil {
		return false, err
	}
	if t.SDHash != sdHash {
		return false, fmt.Errorf("stream mismatch: requested %v, token valid for %v", sdHash, t.SDHash)
	}
	return true, nil
}

func (k *pubKeyManager) loadFromBytes(b []byte) error {
	block, _ := pem.Decode(b)
	if block == nil {
		return errors.New("no PEM blob found")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	key := pubKey.(*rsa.PublicKey)
	if err != nil {
		return err
	}
	k.key = key
	return nil
}

// ValidateToken parses a setialized JWS stream token, verifies its signature, expiry date and returns StreamToken
func (k *pubKeyManager) ValidateToken(stringToken string) (*StreamToken, error) {
	token, err := jwt.ParseWithClaims(stringToken, &StreamToken{}, func(token *jwt.Token) (interface{}, error) {
		return k.key, nil
	})
	if err != nil {
		return nil, err
	}
	if streamToken, ok := token.Claims.(*StreamToken); ok && token.Valid {
		return streamToken, nil
	}
	return nil, err
}
