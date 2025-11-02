package utils

import "crypto/sha256"

// SumSHA256 returns the SHA-256 checksum of the provided data.
func SumSHA256(data []byte) [32]byte {
	return sha256.Sum256(data)
}
 

