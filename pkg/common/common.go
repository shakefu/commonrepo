// Package common contains things which are used across other packages and need
// to be extracted to prevent import cycles
package common

import (
	"os"
	"sort"
)

// ConfigFileGlob is a glob pattern for locating a repo's config
var defaults struct {
	ConfigFileGlob string
}

func init() {
	defaults.ConfigFileGlob = ".commonrepo.{yaml,yml}"
}

// ConfigFileGlob returns the glob pattern for locating a repo's config
func ConfigFileGlob() string {
	if val, ok := os.LookupEnv("COMMON_CONFIG_GLOB"); ok {
		return val
	}
	return defaults.ConfigFileGlob
}

// SortedKeys returns the keys of the given map in sorted order
func SortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ShortestKey returns the shortest key from the map
func ShortestKey(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i int, j int) bool {
		return len(keys[i]) < len(keys[j])
	})
	return keys[0]
}
