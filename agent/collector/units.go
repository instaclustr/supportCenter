package collector

import "fmt"

var decimapAbbrs = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

// HumanSize returns a human-readable approximation of a size
// capped at 4 valid numbers (eg. "2.746 MB", "796 KB").
func HumanSize(size float64) string {
	return customSize("%.4g %s", size, 1000.0, decimapAbbrs)
}

// CustomSize returns a human-readable approximation of a size
// using custom format.
func customSize(format string, size float64, base float64, _map []string) string {
	i := 0
	for size >= base {
		size = size / base
		i++
	}
	return fmt.Sprintf(format, size, _map[i])
}
