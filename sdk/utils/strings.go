package sdkutils

import (
	"crypto/sha1"
	"encoding/hex"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Coverts string into int, returning defaultval if the provided string is not convertable or if an error occur
func AtoiOrDefault(i string, defaultval int) int {
	result, err := strconv.Atoi(i)
	if err != nil {
		return defaultval
	}
	return result
}

// Returns random string with "n" length
func RandomStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// Returns the sha1 sum of all strings
func Sha1Hash(texts ...string) string {
	allstr := strings.Join(texts, "")
	hash := sha1.Sum([]byte(allstr))
	return hex.EncodeToString(hash[:])
}

// Returns a slugged version of a string
func Slugify(input string, separator string) string {
	if separator == "" {
		separator = "_"
	}

	// Convert to lowercase
	result := strings.ToLower(input)

	// Remove special characters
	re := regexp.MustCompile("[^a-z0-9]+")
	result = re.ReplaceAllString(result, separator)

	// Remove leading and trailing hyphens
	result = strings.Trim(result, separator)

	return result
}

// Remove characters from a string
func TrimChars(str string, chars ...string) string {
	for _, c := range chars {
		str = strings.Trim(str, c)
	}
	return str
}
