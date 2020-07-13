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
		var manifest shared.Manifest
		err := json.Unmarshal(data, &manifest)
		if err != nil {
			panic(err)
		}

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
							fullPath := path.Join(fileDir, result.TestPath)
							var ok bool
							var err error

							assert.False(t, strings.Contains(result.TestPath, "/"),
								"%s is not the correct directory for %s", fileDir, result.TestPath)
							if result.TestPath == "*" {
								ok, err = manifest.ContainsFile(fileDir)
							} else {
								ok, err = manifest.ContainsTest(fullPath)
							}
							assert.Nil(t, err)
							assert.True(t, ok,
								"%s is not a test path found in the manifest @ %s", fullPath, sha)
						})
					}
				}
			})
			return nil
		})
	})
}
