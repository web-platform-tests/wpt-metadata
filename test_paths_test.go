// Copyright 2019 The WPT Dashboard Project. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metadata

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
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

// TestResultsTestPaths enumerates all the metadata in the repo and ensures that
// a test by the given name exists in the latest manifest.
func TestResultsTestPaths(t *testing.T) {
	log.Println("Fetching latest manifest...")
	resp, err := http.Get("https://wpt.fyi/api/manifest?sha=latest")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	sha := resp.Header.Get("x-wpt-sha")
	if sha == "" {
		sha = shared.LatestSHA
	} else {
		sha = sha[:7]
	}

	t.Run(fmt.Sprintf("Manifest @ %s", sha), func(t *testing.T) {
		unmarshalled := shared.Manifest{}
		err := json.Unmarshal(data, &unmarshalled)
		if err != nil {
			panic(err)
		}

		validPaths := mapset.NewSet()
		manifestItems := []shared.ManifestItem{
			unmarshalled.Items.Manual,
			unmarshalled.Items.Reftest,
			unmarshalled.Items.TestHarness,
			unmarshalled.Items.WDSpec,
		}
		for _, m := range manifestItems {
			if m == nil {
				continue
			}
			for _, items := range m {
				for _, item := range items {
					var url string
					if err = json.Unmarshal(*item[0], &url); err != nil {
						continue
					}
					validPaths.Add(url)
				}
			}
		}
		log.Printf("Found %v test paths", validPaths.Cardinality())

		// Crawl + test all the metadata
		filepath.Walk(".", func(filePath string, info os.FileInfo, err error) error {
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
				data, err := ioutil.ReadAll(f)
				if err != nil {
					panic(err)
				}
				err = yaml.Unmarshal(data, &metadata)
				assert.Nil(t, err)

				for _, link := range metadata.Links {
					for _, result := range link.Results {
						// TODO: support testing wildcard (*) TestPaths
						if result.TestPath != "" && !strings.HasSuffix(result.TestPath, "*") {
							fullPath := path.Join(fileDir, result.TestPath)
							t.Run(fullPath, func(t *testing.T) {
								assert.True(
									t,
									validPaths.Contains(fullPath),
									"%s is not a test path found in the manifest @ %s", fullPath, sha,
								)
							})
						}
					}
				}
			})
			return nil
		})
	})
}
