package api

import (
	"strconv"
	"strings"
	"time"
)

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}
