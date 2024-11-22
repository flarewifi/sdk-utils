package sdkstr

import "strconv"

// Coverts string into int, returning defaultval if the provided string is not convertable or if an error occur
func AtoiOrDefault(i string, defaultval int) int {
	result, err := strconv.Atoi(i)
	if err != nil {
		return defaultval
	}
	return result
}
