// Package common provides shared cryptographic utilities
package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

const (
	KeySize = 32 // WireGuard uses Curve25519 keys
)

// GeneratePrivateKey generates a new WireGuard private key
func GeneratePrivateKey() (string, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Clamp the key for Curve25519
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64

	return base64.StdEncoding.EncodeToString(key), nil
}

// GetPublicKey derives the public key from a private key
func GetPublicKey(privateKeyBase64 string) (string, error) {
	privKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privKey) != KeySize {
		return "", fmt.Errorf("invalid private key length: %d, expected %d", len(privKey), KeySize)
	}

	var pubKey [KeySize]byte
	var privKeyArray [KeySize]byte
	copy(privKeyArray[:], privKey)

	curve25519.ScalarBaseMult(&pubKey, &privKeyArray)

	return base64.StdEncoding.EncodeToString(pubKey[:]), nil
}

// GenerateKeyPair generates a new private/public key pair
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	privateKey, err = GeneratePrivateKey()
	if err != nil {
		return "", "", err
	}

	publicKey, err = GetPublicKey(privateKey)
	if err != nil {
		return "", "", err
	}

	return privateKey, publicKey, nil
}

// ValidateKey validates that a key is a valid base64-encoded 32-byte key
func ValidateKey(keyBase64 string) error {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	if len(key) != KeySize {
		return fmt.Errorf("invalid key length: %d, expected %d", len(key), KeySize)
	}

	return nil
}
