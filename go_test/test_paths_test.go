// Copyright 2019 The WPT Dashboard Project. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package go_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

var (
	liveManifestURL  = "https://wpt.fyi/api/manifest?sha=latest"
	httpClient       = &http.Client{}
	liveManifestOnce sync.Once
	liveManifest     *shared.Manifest
	liveManifestErr  error
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func isLiveManifestFallbackEnabled() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("CI"))) == "true"
}

func getLiveManifest() (*shared.Manifest, error) {
	liveManifestOnce.Do(func() {
		log.Println("[NOTICE] Retrieving live WPT manifest from", liveManifestURL, "(CI=true)...")
		resp, err := httpClient.Get(liveManifestURL)
		if err != nil {
			liveManifestErr = fmt.Errorf("failed to fetch live manifest: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			liveManifestErr = fmt.Errorf("unexpected HTTP status code %d from live manifest endpoint", resp.StatusCode)
			return
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			liveManifestErr = fmt.Errorf("failed to read live manifest: %w", err)
			return
		}

		var m shared.Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			liveManifestErr = fmt.Errorf("failed to unmarshal live manifest: %w", err)
			return
		}
		liveManifest = &m
	})
	return liveManifest, liveManifestErr
}

// TestResultsTestPaths enumerates all the metadata in the repo and ensures that
// a test by the given name exists in the latest manifest.
func TestResultsTestPaths(t *testing.T) {
	wpt_manifest, err := os.Open("MANIFEST.json")
	if err != nil {
		panic(err)
	}
	defer wpt_manifest.Close()

	data, err := ioutil.ReadAll(wpt_manifest)
	if err != nil {
		panic(err)
	}

	t.Run(fmt.Sprintf("Run against WPT's MANIFEST.json"), func(t *testing.T) {
		var manifest shared.Manifest
		err := json.Unmarshal(data, &manifest)
		if err != nil {
			panic(err)
		}

		// Crawl + test all the metadata
		filepath.Walk("../", func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				panic(err)
			} else if info.IsDir() || strings.ToLower(info.Name()) != "meta.yml" {
				return nil
			}
			fileDir := path.Dir(filePath)
			t.Run(fileDir, func(t *testing.T) {
				var metadata shared.Metadata
				f, err := os.Open(filePath)
				if err != nil {
					panic(err)
				}
				defer f.Close()

				data, err := ioutil.ReadAll(f)
				if err != nil {
					panic(err)
				}
				err = yaml.Unmarshal(data, &metadata)
				assert.Nil(t, err)

				seen := mapset.NewSet()
				for _, link := range metadata.Links {
					for _, result := range link.Results {
						if result.TestPath == "" || seen.Contains(result.TestPath) {
							continue
						}
						seen.Add(result.TestPath)

						t.Run(result.TestPath, func(t *testing.T) {
							absFileDir := strings.TrimLeft(fileDir, "../")
							fullPath := path.Join(absFileDir, result.TestPath)
							var ok bool
							var err error
							if result.TestPath == "*" {
								ok, err = manifest.ContainsFile(absFileDir)
							} else {
								ok, err = manifest.ContainsTest(fullPath)
							}
							if !ok || err != nil {
								if isLiveManifestFallbackEnabled() {
									log.Printf("[NOTICE] Test path %q not found in local MANIFEST.json. Attempting live manifest check (enabled via CI=true)...", fullPath)
									if liveM, lErr := getLiveManifest(); lErr == nil && liveM != nil {
										if result.TestPath == "*" {
											ok, err = liveM.ContainsFile(absFileDir)
										} else {
											ok, err = liveM.ContainsTest(fullPath)
										}
									}
								} else {
									log.Printf("[NOTICE] Test path %q not found in local MANIFEST.json. Live manifest fallback is disabled. To check against live upstream WPT manifest locally, run with: CI=true go test ./...", fullPath)
								}
							}
							assert.Nil(t, err)
							assert.True(t, ok,
								"%s is not a test path found in the manifest", fullPath)
						})
					}
				}
			})
			return nil
		})
	})
}

func TestLiveManifestFallback_Mock(t *testing.T) {
	mockJSON := `{"version": 8, "items": {"testharness": {"mock": {"new-test.html": ["abcdef1234567890", [null, {}]]}}}}`
	origTransport := httpClient.Transport
	httpClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, liveManifestURL, req.URL.String())
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(mockJSON)),
		}, nil
	})

	liveManifestOnce = sync.Once{}
	liveManifest = nil
	liveManifestErr = nil
	defer func() {
		httpClient.Transport = origTransport
		liveManifestOnce = sync.Once{}
		liveManifest = nil
		liveManifestErr = nil
	}()

	liveM, err := getLiveManifest()
	assert.Nil(t, err)
	assert.NotNil(t, liveM)
	ok, err := liveM.ContainsTest("mock/new-test.html")
	assert.Nil(t, err)
	assert.True(t, ok)
}

func TestLiveManifestFallback_LiveNetwork(t *testing.T) {
	liveManifestOnce = sync.Once{}
	liveManifest = nil
	liveManifestErr = nil
	defer func() {
		liveManifestOnce = sync.Once{}
		liveManifest = nil
		liveManifestErr = nil
	}()

	liveM, err := getLiveManifest()
	if err != nil {
		t.Skipf("Skipping live network verification due to network/endpoint error or non-200 status: %v", err)
		return
	}
	assert.NotNil(t, liveM)
	ok, err := liveM.ContainsTest("xhr/send-redirect.htm")
	assert.Nil(t, err)
	assert.True(t, ok)
}

func TestIsLiveManifestFallbackEnabled(t *testing.T) {
	t.Setenv("CI", "true")
	assert.True(t, isLiveManifestFallbackEnabled())

	t.Setenv("CI", "false")
	assert.False(t, isLiveManifestFallbackEnabled())
}

