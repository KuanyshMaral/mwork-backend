package robokassa

import "strings"

// sortStrings sorts a slice of strings in place (bubble sort)
// Used for RoboKassa parameter ordering
func sortStrings(slice []string) {
	for i := 0; i < len(slice)-1; i++ {
		for j := i + 1; j < len(slice); j++ {
			if slice[i] > slice[j] {
				slice[i], slice[j] = slice[j], slice[i]
			}
		}
	}
}

// getFirstValue extracts the first value from form values (case-insensitive lookup)
// Used for RoboKassa webhook form parsing
func getFirstValue(values map[string][]string, key string) string {
	for k, v := range values {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}
