// Copyright 2019 The WPT Dashboard Project. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package go_test

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/web-platform-tests/wpt.fyi/shared"
)

func TestParseMetadata(t *testing.T) {
	filepath.Walk("../", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		} else if info.IsDir() || strings.ToLower(info.Name()) != "meta.yml" {
			return nil
		}
		t.Run(path, func(t *testing.T) {
			var metadata shared.Metadata
			f, err := os.Open(path)
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

			linkCount := strings.Count(string(data), "links:")
			assert.Equal(t, 1, linkCount, "YML file should contain exactly one links: key")
			assert.Greater(t, len(metadata.Links), 0)
			linkMap := make(map[string]string)
			labelSet := mapset.NewSet()
			for _, link := range metadata.Links {
				// Check if it is a test-level link,
				if link.Product.String() == "" {
					checkTestLevelLinks(t, link)
				}

				if link.URL != "" {
					_, err := url.ParseRequestURI(link.URL)
					assert.Nil(t, err)
				} else {
					assert.True(t, link.Label != "", "url and label cannot both be empty.")
				}

				if link.Label != "" {
					// Check duplicated labels.
					assert.False(t, labelSet.Contains(link.Label), fmt.Sprintf("label %s already exists", link.Label))
					labelSet.Add(link.Label)
					assert.True(t, link.Product.String() == "", "label is present at the test-level only.")
				}
				checkDuplicationAcrossLinks(t, link, linkMap)

				assert.Greater(t, len(link.Results), 0)
				resultSet := mapset.NewSet()
				for _, result := range link.Results {
					assert.Greater(t, len(result.TestPath), 0)
					assert.False(t, strings.Contains(result.TestPath, "/"), "Test files must not be paths")
					checkDuplicationWithinResults(t, result, resultSet, path)
				}
			}
		})
		return nil
	})
}

func checkDuplicationAcrossLinks(t *testing.T, link shared.MetadataLink, linkMap map[string]string) {
	val, ok := linkMap[link.URL]
	expected := serializeStrings(link.Product.String())
	if ok {
		assert.NotEqual(t, val, expected, "duplicated entries between two links")
	}
	linkMap[link.URL] = expected
}

func checkDuplicationWithinResults(t *testing.T, result shared.MetadataTestResult, resultSet mapset.Set, path string) {
	subtestName := ""
	if result.SubtestName != nil {
		subtestName = *result.SubtestName
	}
	status := ""
	if result.Status != nil {
		status = result.Status.String()
	}
	expected := serializeStrings(result.TestPath, subtestName, status)
	assert.False(t, resultSet.Contains(expected), fmt.Sprintf("duplicated entries within results: %s in %s", result.TestPath, path))
	resultSet.Add(expected)
}

func serializeStrings(val ...string) string {
	var returnVal string
	for i := 0; i < len(val); i++ {
		returnVal += strings.TrimSpace(val[i])
	}

	return returnVal
}

func checkTestLevelLinks(t *testing.T, link shared.MetadataLink) {
	// Subtest should be empty for a test-level link.
	for _, result := range link.Results {
		assert.Nil(t, result.SubtestName, fmt.Sprintf("subtest should be empty for a test-level link"))
	}
}
