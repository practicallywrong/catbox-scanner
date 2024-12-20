package utils

import (
	"math/rand"
)

func GenerateRandomID(size int, charset string) string {
	id := make([]byte, size)
	for i := range id {
		id[i] = charset[rand.Intn(len(charset))]
	}
	return string(id)
}
