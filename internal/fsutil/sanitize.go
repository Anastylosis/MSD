package fsutil

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxFilenameBytes = 255

var windowsReserved = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[0-9]|LPT[0-9])(\.|$)`)

func SanitizeName(name string) string {
	if name == "" {
		return "_"
	}

	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "\x00", "")

	for _, c := range []string{"<", ">", ":", "\"", "|", "?", "*"} {
		name = strings.ReplaceAll(name, c, "_")
	}

	name = strings.TrimLeft(name, ". ")
	name = strings.TrimRight(name, ". ")

	if windowsReserved.MatchString(name) {
		name = "_" + name
	}

	name = truncateToBytes(name, maxFilenameBytes)

	if name == "" {
		return "_"
	}

	return name
}

func SanitizePath(name string) string {
	return SanitizeName(name)
}

func truncateToBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	ext := filepath.Ext(s)
	if len(ext) > 0 && len(ext) < maxBytes-10 {
		base := s[:len(s)-len(ext)]
		base = truncateUTF8(base, maxBytes-len(ext))
		return base + ext
	}
	return truncateUTF8(s, maxBytes)
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
