// Copyright 2019 The WPT Dashboard Project. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metadata

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func TestParseMetadata(t *testing.T) {
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
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
			data, err := ioutil.ReadAll(f)
			if err != nil {
				panic(err)
			}
			err = yaml.Unmarshal(data, &metadata)
			assert.Nil(t, err)

			linkCount := strings.Count(string(data), "links:")
			assert.Equal(t, linkCount, 1, "YML file should only contain one `links:` key")
			assert.Greater(t, len(metadata.Links), 0)
			linkMap := make(map[string]string)
			for _, link := range metadata.Links {
				assert.Greater(t, len(link.Results), 0)
				assert.Greater(t, len(link.URL), 0)
				checkDuplicationAcrossLinks(t, link, linkMap)
				resultSet := mapset.NewSet()
				for _, result := range link.Results {
					assert.Greater(t, len(result.TestPath), 0)
					checkDuplicationWithinResults(t, result, resultSet)
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

func checkDuplicationWithinResults(t *testing.T, result shared.MetadataTestResult, resultSet mapset.Set) {
	subtestName := ""
	if result.SubtestName != nil {
		subtestName = *result.SubtestName
	}
	status := ""
	if result.Status != nil {
		status = result.Status.String()
	}
	expected := serializeStrings(result.TestPath, subtestName, status)
	assert.False(t, resultSet.Contains(expected), "duplicated entries within results")
	resultSet.Add(expected)
}

func serializeStrings(val ...string) string {
	var returnVal string
	for i := 0; i < len(val); i++ {
		returnVal += strings.TrimSpace(val[i])
	}

	return returnVal
}
