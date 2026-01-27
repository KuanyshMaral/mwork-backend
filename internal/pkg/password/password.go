package password

import (
	"golang.org/x/crypto/bcrypt"
)

const cost = 12 // bcrypt cost factor (higher = slower but more secure)

// Hash hashes password using bcrypt
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

// Verify compares password with hash
func Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
