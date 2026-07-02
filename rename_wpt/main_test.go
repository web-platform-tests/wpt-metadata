package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func TestSweepStaleMetadataAgainstManifest(t *testing.T) {
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tempDir)
	assert.NoError(t, err)
	defer os.Chdir(origDir)

	err = os.MkdirAll("testdir", 0755)
	assert.NoError(t, err)

	metaPath := filepath.Join("testdir", "META.yml")
	initialYAML := []byte(`links:
  - product: chrome
    url: https://external.com/item
    results:
    - test: valid.html
    - test: stale.html
`)
	err = os.WriteFile(metaPath, initialYAML, 0644)
	assert.NoError(t, err)

	mockManifestJSON := []byte(`{
		"version": 8,
		"items": {
			"testharness": {
				"testdir": {
					"valid.html": ["abcdef1234567890", [null, {}]]
				}
			}
		}
	}`)
	var mockManifest shared.Manifest
	err = json.Unmarshal(mockManifestJSON, &mockManifest)
	assert.NoError(t, err)

	sweepStaleMetadataAgainstManifest(&mockManifest)

	finalBytes, err := os.ReadFile(metaPath)
	assert.NoError(t, err)
	finalStr := string(finalBytes)

	assert.Contains(t, finalStr, "valid.html")
	assert.NotContains(t, finalStr, "stale.html")
}
