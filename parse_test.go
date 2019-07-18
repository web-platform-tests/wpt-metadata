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
			if strings.Contains(string(data), "links:") {
				assert.Greater(t, len(metadata.Links), 0)
				for _, link := range metadata.Links {
					assert.Greater(t, len(link.Results), 0)
				}
			}
		})
		return nil
	})
}
