package config

import "strings"

func prefixMatchesOverride(prefix, overrideKeyPrefix string) bool {
	normalizedPrefix := strings.Trim(prefix, "/")
	normalizedOverride := strings.Trim(overrideKeyPrefix, "/")
	if normalizedOverride == "" {
		return true
	}
	return normalizedPrefix == normalizedOverride ||
		strings.HasPrefix(normalizedPrefix, normalizedOverride+"/")
}
