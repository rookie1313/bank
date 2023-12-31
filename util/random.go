package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	alphabet = "abcdefghijhlmnopqrstuvwxyz"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomInt generates random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

// RandomString generates random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)
	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomOwner generates random owner name(length:6)
func RandomOwner() string {
	return RandomString(6)
}

// RandomMoney generates random money
func RandomMoney() int64 {
	return RandomInt(0, 100000)
}

func RandomCurrency() string {
	currencies := []string{USD, CAD, EUR}
	n := len(currencies)
	return currencies[rand.Intn(n)]
}

// RandomEmail generates random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}
