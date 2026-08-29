// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package update

import (
	"strings"

	"golang.org/x/mod/semver"
)

// NormalizeVersion strips a leading "v"/"V" and surrounding whitespace.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	return strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
}

// CanonicalTag returns a GitHub-style tag with a leading "v".
func CanonicalTag(v string) string {
	n := NormalizeVersion(v)
	if n == "" {
		return ""
	}
	return "v" + n
}

func toSemver(v string) string {
	n := NormalizeVersion(v)
	if n == "" || n == "dev" {
		return ""
	}
	return "v" + n
}

// IsValid reports whether v parses as a semver (with optional pre-release).
func IsValid(v string) bool {
	return semver.IsValid(toSemver(v))
}

// Compare returns -1, 0, or 1 as a is less than, equal to, or greater than b.
// Invalid / "dev" versions compare as older than valid ones.
func Compare(a, b string) int {
	sa, sb := toSemver(a), toSemver(b)
	if sa == "" && sb == "" {
		return strings.Compare(NormalizeVersion(a), NormalizeVersion(b))
	}
	if sa == "" {
		return -1
	}
	if sb == "" {
		return 1
	}
	return semver.Compare(sa, sb)
}

// IsNewer reports whether candidate is strictly newer than current.
func IsNewer(candidate, current string) bool {
	return Compare(candidate, current) > 0
}
