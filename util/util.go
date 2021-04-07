package util

import (
	"fmt"
	"strings"
)

// FormatBytes takes an int and returns its formatted form, example 5235745682 -> "5.2 GB"
func FormatBytes(b int) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

// IsURL checks to see if a string is a url or a torrent file path
func IsURL(url string) bool {
	return strings.HasPrefix(url, "magnet:") || strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
