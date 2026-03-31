package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

func SanitizeName(name string) string {
	// Remove anything that isn't alphanumeric or safe characters to prevent path traversal
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	return reg.ReplaceAllString(name, "_")
}

func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".cache", "salat-break")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

func Save(name string, data interface{}) error {
	dir, err := getCacheDir()
	if err != nil {
		return err
	}
	
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filepath.Join(dir, name), bytes, 0644)
}

func Load(name string, target interface{}) error {
	dir, err := getCacheDir()
	if err != nil {
		return err
	}
	
	bytes, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	
	return json.Unmarshal(bytes, target)
}

func GetModTime(name string) (time.Time, error) {
	dir, err := getCacheDir()
	if err != nil {
		return time.Time{}, err
	}
	
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		return time.Time{}, err
	}
	
	return info.ModTime(), nil
}
