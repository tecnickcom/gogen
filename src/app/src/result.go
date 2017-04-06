package main

import (
	"crypto/rand"
	"math/big"
)

// getResult returns the result
// NOTE: This is just a dummy example function
func getResult() string {
	length := 32
	charset := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	password := make([]byte, length) // #nosec
	chars := []byte(charset)
	maxValue := new(big.Int).SetInt64(int64(len(charset)))

	for i := 0; i < length; i++ {
		rnd, err := rand.Int(rand.Reader, maxValue)
		if err == nil {
			password[i] = chars[rnd.Int64()]
		}
	}

	return string(password)
}
