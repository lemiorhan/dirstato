package units

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var suffixes = []string{"B", "K", "M", "G", "T", "P"}

var sizePattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([kmgtp]?)(?:i?b)?$`)

func ParseSize(s string) (int64, error) {
	m := sizePattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(s)))
	if m == nil {
		return 0, fmt.Errorf("geçersiz boyut: %q (örnek: 500M, 1.5G, 0)", s)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("geçersiz boyut: %q", s)
	}
	exp := strings.Index("kmgtp", m[2]) + 1
	if m[2] == "" {
		exp = 0
	}
	bytes := v * math.Pow(1024, float64(exp))
	if bytes >= math.MaxInt64 {
		return 0, fmt.Errorf("boyut çok büyük: %q", s)
	}
	return int64(bytes), nil
}

func FormatSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	v := float64(n)
	i := 0
	for v >= 1024 && i < len(suffixes)-1 {
		v /= 1024
		i++
	}
	// %.1f rounds 1023.95 up to "1024.0"; promote so the unit stays below 1024.
	if s := fmt.Sprintf("%.1f", v); s == "1024.0" && i < len(suffixes)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f%s", v, suffixes[i])
}
