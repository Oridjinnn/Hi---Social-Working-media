package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CacheFile[T any] struct {
	Path     string
	TTL      time.Duration
	LastRead time.Time
	Value    T
}

// ReadJSONCache reads JSON from `path` and validates TTL.
// If file does not exist or is stale, it returns (zero, false, nil).
func ReadJSONCache[T any](path string, ttl time.Duration) (T, bool, error) {
	var zero T

	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("stat cache: %w", err)
	}
	if ttl > 0 && time.Since(st.ModTime()) > ttl {
		return zero, false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return zero, false, fmt.Errorf("read cache: %w", err)
	}

	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return zero, false, fmt.Errorf("unmarshal cache: %w", err)
	}
	return out, true, nil
}

func WriteJSONCache[T any](path string, v T) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir cache dir: %w", err)
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
