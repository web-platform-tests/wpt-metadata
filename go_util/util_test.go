package go_util

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"gopkg.in/yaml.v3"
)

func TestAppendMetadataLink_AddNewLink(t *testing.T) {
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

	appendMetadataLink("abc", amendment, &metadata)

	assert.Equal(t, 3, len(metadata.Links))
	assert.Equal(t, "chrome", metadata.Links[2].Product.BrowserName)
	assert.Equal(t, "bugs.bar?id=456", metadata.Links[2].URL)
	assert.Equal(t, 1, len(metadata.Links[2].Results))
	assert.Equal(t, "d.html", metadata.Links[2].Results[0].TestPath)
}

func TestAppendMetadataLink_AddNewResult(t *testing.T) {
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

	appendMetadataLink("abc", amendment, &metadata)

	assert.Equal(t, 2, len(metadata.Links))
	assert.Equal(t, "chrome", metadata.Links[0].Product.BrowserName)
	assert.Equal(t, "https://external.com/item", metadata.Links[0].URL)
	assert.Equal(t, 2, len(metadata.Links[0].Results))
	assert.Equal(t, "a.html", metadata.Links[0].Results[0].TestPath)
	assert.Equal(t, "d.html", metadata.Links[0].Results[1].TestPath)
}

func TestDeleteMetadata_NoDeletion(t *testing.T) {
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

	_, resultLinks := deleteMetadata("abc.html", metadata)

	assert.Equal(t, 0, len(resultLinks))
}

func TestDeleteMetadata_DeletedOneLink(t *testing.T) {
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

	resultMetadata, resultLinks := deleteMetadata("a.html", metadata)

	assert.Equal(t, 1, len(resultLinks))
	assert.Equal(t, "chrome", resultLinks[0].Product.BrowserName)
	assert.Equal(t, "https://external.com/item", resultLinks[0].URL)
	assert.Equal(t, 1, len(resultLinks[0].Results))
	assert.Equal(t, "a.html", resultLinks[0].Results[0].TestPath)

	assert.Equal(t, 1, len(resultMetadata.Links))
	assert.Equal(t, "firefox", resultMetadata.Links[0].Product.BrowserName)
	assert.Equal(t, "https://bug.com/item", resultMetadata.Links[0].URL)
	assert.Equal(t, 2, len(resultMetadata.Links[0].Results))
	assert.Equal(t, "b.html", resultMetadata.Links[0].Results[0].TestPath)
	assert.Equal(t, "c.html", resultMetadata.Links[0].Results[1].TestPath)
}

func TestDeleteMetadata_DeletedMultipleLinks(t *testing.T) {
	fileInBytes := []byte(`
links:
  - product: chrome
    url: https://external.com/item
    results:
    - test: a.html
  - product: firefox
    url: https://bug.com/item
    results:
    - test: a.html
      subtest: Something should happen
      status: FAIL
    - test: c.html
`)
	var metadata shared.Metadata
	yaml.Unmarshal(fileInBytes, &metadata)

	resultMetadata, resultLinks := deleteMetadata("a.html", metadata)

	assert.Equal(t, 2, len(resultLinks))
	assert.Equal(t, "chrome", resultLinks[0].Product.BrowserName)
	assert.Equal(t, "https://external.com/item", resultLinks[0].URL)
	assert.Equal(t, 1, len(resultLinks[0].Results))
	assert.Equal(t, "a.html", resultLinks[0].Results[0].TestPath)
	assert.Equal(t, "firefox", resultLinks[1].Product.BrowserName)
	assert.Equal(t, "https://bug.com/item", resultLinks[1].URL)
	assert.Equal(t, 1, len(resultLinks[0].Results))
	assert.Equal(t, "a.html", resultLinks[0].Results[0].TestPath)

	assert.Equal(t, 1, len(resultMetadata.Links))
	assert.Equal(t, "firefox", resultMetadata.Links[0].Product.BrowserName)
	assert.Equal(t, "https://bug.com/item", resultMetadata.Links[0].URL)
	assert.Equal(t, 1, len(resultMetadata.Links[0].Results))
	assert.Equal(t, "c.html", resultMetadata.Links[0].Results[0].TestPath)
}

func TestGetYMLFilePath(t *testing.T) {
	assert.Equal(t, "abc/META.yml", getYMLFilePath("abc/test.html"))
}
