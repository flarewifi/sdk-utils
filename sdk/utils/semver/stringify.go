package sdksemver

import (
	"fmt"
)

// Returns a string version with format v<major>.<minor>.<patch>
func StringifyVersion(data Version) string {
	return fmt.Sprintf("v%v.%v.%v", data.Major, data.Minor, data.Patch)
}
