package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.json", "normal.json"},
		{"space name.json", "space_name.json"},
		{"../../etc/passwd", ".._.._etc_passwd"},
		{"@#$%^&*()", "_"},
		{"with-dash_and.dot", "with-dash_and.dot"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, SanitizeName(tt.input))
		})
	}
}

func TestCache_SaveAndLoad(t *testing.T) {
	// Setup temp dir for cache
	tmpDir, err := os.MkdirTemp("", "salat-break-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	BaseDir = tmpDir

	type TestData struct {
		Name  string
		Value int
	}

	data := TestData{Name: "test", Value: 123}
	filename := "test_cache.json"

	// Test Save
	err = Save(filename, data)
	assert.NoError(t, err)

	// Verify file exists
	cacheDir := filepath.Join(tmpDir, ".cache", "salat-break")
	_, err = os.Stat(filepath.Join(cacheDir, filename))
	assert.NoError(t, err)

	// Test Load
	var loaded TestData
	err = Load(filename, &loaded)
	assert.NoError(t, err)
	assert.Equal(t, data, loaded)

	// Test GetModTime
	modTime, err := GetModTime(filename)
	assert.NoError(t, err)
	assert.False(t, modTime.IsZero())

	// Test Delete
	err = Delete(filename)
	assert.NoError(t, err)

	// Verify file is gone
	_, err = os.Stat(filepath.Join(cacheDir, filename))
	assert.True(t, os.IsNotExist(err))
}

func TestLoad_NonExistent(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "salat-break-test-*")
	defer os.RemoveAll(tmpDir)
	BaseDir = tmpDir

	var data map[string]string
	err := Load("non_existent.json", &data)
	assert.Error(t, err)
}
