package sdksemver

type Version struct {
	Major int
	Minor int
	Patch int
}

// Checks if the current version is behind from latest version.
// Returns true if current >= to latest, otherwise false
func HasUpdates(current, latest Version) bool {
	if current.Major < latest.Major {
		return true
	}
	if current.Minor < latest.Minor {
		return true
	}
	if current.Patch < latest.Patch {
		return true
	}
	return false
}
