package main

import (
	"os"
	"strconv"
)

func envOrDefaultString(key, s string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return s
}

func envOrDefaultInt(key string, n int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return n
		}
		return i
	}
	return n
}

func envOrDefaultBool(key string, b bool) bool {
	if v := os.Getenv(key); v != "" {
		b, _ = strconv.ParseBool(v)
		return b
	}
	return b
}
