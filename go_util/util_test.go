package go_util

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"gopkg.in/yaml.v3"
)

func TestAppendPendingMetadata_AddNewLink(t *testing.T) {
	var amendment shared.MetadataLink
	json.Unmarshal([]byte(`
		{
			"url": "bugs.bar?id=456",
			"product": "chrome",
			"results": [
				{"test": "d.html"}
			]
		}
	`), &amendment)

	fileInBytes := []byte(`
links:
  - product: chrome
    url: https://external.com/item
    results:
    - test: a.html
  - product: firefox
    url: https://bug.com/item
    results:
    - test: b.html
      subtest: Something should happen
      status: FAIL
    - test: c.html
`)
	var metadata shared.Metadata
	yaml.Unmarshal(fileInBytes, &metadata)

	AppendPendingMetadata("abc", amendment, &metadata)

	assert.Equal(t, 3, len(metadata.Links))
	assert.Equal(t, "chrome", metadata.Links[2].Product.BrowserName)
	assert.Equal(t, "bugs.bar?id=456", metadata.Links[2].URL)
	assert.Equal(t, 1, len(metadata.Links[2].Results))
	assert.Equal(t, "d.html", metadata.Links[2].Results[0].TestPath)
}

func TestAppendPendingMetadata_AddNewResult(t *testing.T) {
	var amendment shared.MetadataLink
	json.Unmarshal([]byte(`
		{
			"url": "https://external.com/item",
			"product": "chrome",
			"results": [
				{"test": "d.html"}
			]
		}
	`), &amendment)

	fileInBytes := []byte(`
links:
  - product: chrome
    url: https://external.com/item
    results:
    - test: a.html
  - product: firefox
    url: https://bug.com/item
    results:
    - test: b.html
      subtest: Something should happen
      status: FAIL
    - test: c.html
`)
	var metadata shared.Metadata
	yaml.Unmarshal(fileInBytes, &metadata)

	AppendPendingMetadata("abc", amendment, &metadata)

	assert.Equal(t, 2, len(metadata.Links))
	assert.Equal(t, "chrome", metadata.Links[0].Product.BrowserName)
	assert.Equal(t, "https://external.com/item", metadata.Links[0].URL)
	assert.Equal(t, 2, len(metadata.Links[0].Results))
	assert.Equal(t, "a.html", metadata.Links[0].Results[0].TestPath)
	assert.Equal(t, "d.html", metadata.Links[0].Results[1].TestPath)
}
