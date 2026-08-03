package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	childPINTime    = 2
	childPINMemory  = 32 * 1024
	childPINThreads = 1
	childPINKeyLen  = 32
)

func HashChildPIN(pin string) (string, error) {
	if !validChildPIN(pin) {
		return "", errors.New("PIN must contain exactly four digits")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate child PIN salt: %w", err)
	}
	hash := argon2.IDKey([]byte(pin), salt, childPINTime, childPINMemory, childPINThreads, childPINKeyLen)
	return fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", childPINMemory, childPINTime, childPINThreads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func VerifyChildPIN(encoded, pin string) bool {
	if !validChildPIN(pin) {
		return false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return false
	}
	var memory uint32
	var timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false
	}
	if memory == 0 || memory > 128*1024 || timeCost == 0 || timeCost > 8 || threads == 0 || threads > 4 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(salt) < 8 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(expected) != childPINKeyLen {
		return false
	}
	actual := argon2.IDKey([]byte(pin), salt, timeCost, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func validChildPIN(pin string) bool {
	if len(pin) != 4 {
		return false
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
