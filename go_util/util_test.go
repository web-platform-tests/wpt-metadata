package go_util

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"go.yaml.in/yaml/v4"
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
  - label: label1
    results:
    - test: a.html
    - test: b.html
`)
	var metadata shared.Metadata
	yaml.Unmarshal(fileInBytes, &metadata)

	resultMetadata, resultLinks := deleteMetadata("a.html", metadata)

	assert.Equal(t, 3, len(resultLinks))
	assert.Equal(t, "chrome", resultLinks[0].Product.BrowserName)
	assert.Equal(t, "https://external.com/item", resultLinks[0].URL)
	assert.Equal(t, 1, len(resultLinks[0].Results))
	assert.Equal(t, "a.html", resultLinks[0].Results[0].TestPath)
	assert.Equal(t, "firefox", resultLinks[1].Product.BrowserName)
	assert.Equal(t, "https://bug.com/item", resultLinks[1].URL)
	assert.Equal(t, 1, len(resultLinks[1].Results))
	assert.Equal(t, "a.html", resultLinks[1].Results[0].TestPath)
	assert.Equal(t, "label1", resultLinks[2].Label)
	assert.Equal(t, 1, len(resultLinks[2].Results))
	assert.Equal(t, "a.html", resultLinks[1].Results[0].TestPath)

	assert.Equal(t, 2, len(resultMetadata.Links))
	assert.Equal(t, "firefox", resultMetadata.Links[0].Product.BrowserName)
	assert.Equal(t, "https://bug.com/item", resultMetadata.Links[0].URL)
	assert.Equal(t, 1, len(resultMetadata.Links[0].Results))
	assert.Equal(t, "c.html", resultMetadata.Links[0].Results[0].TestPath)
	assert.Equal(t, "label1", resultMetadata.Links[1].Label)
	assert.Equal(t, 1, len(resultMetadata.Links[1].Results))
	assert.Equal(t, "b.html", resultMetadata.Links[1].Results[0].TestPath)
}

func TestGetYMLFilePath(t *testing.T) {
	assert.Equal(t, "abc/META.yml", getYMLFilePath("abc/test.html"))
}

func TestWriteMetadataLink_FileIO_OverwritesCorrectly(t *testing.T) {
	// 1. Set up a temporary test directory
	tempDir := t.TempDir()

	// Create a mock test path inside the temp directory
	testPath := tempDir + "/a.html"
	metaPath := tempDir + "/META.yml"

	// 2. Write an initial META.yml file to disk
	initialYAML := []byte(`links:
  - product: chrome
    url: https://external.com/item
    results:
    - test: a.html
`)
	err := os.WriteFile(metaPath, initialYAML, 0644)
	assert.NoError(t, err)

	// 3. Define the amendment we want to apply
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

	// 4. Execute the function that does the file I/O
	WriteMetadataLink(testPath, amendment)

	// 5. Read the file back from disk
	finalBytes, err := os.ReadFile(metaPath)
	assert.NoError(t, err)
	finalStr := string(finalBytes)

	// ASSERTS

	// Bug 1: Writing Twice (The Original Bug)
	// If the file pointer wasn't reset, "links:" would be written twice.
	assert.Equal(t, 1, strings.Count(finalStr, "links:"), "The file should only contain 'links:' once. Did it append instead of overwrite?")

	// Bug 2: Trailing Garbage (The Hidden WriteAt Bug)
	// If the file wasn't truncated, and the new data was shorter than the old data,
	// trailing bytes would remain, causing YAML parsing to fail.
	var metadata shared.Metadata
	err = yaml.Unmarshal(finalBytes, &metadata)
	assert.NoError(t, err, "File should be valid YAML without trailing garbage bytes")

	// Standard logic checks to ensure the data actually updated
	assert.Equal(t, 1, len(metadata.Links))
	assert.Equal(t, 2, len(metadata.Links[0].Results))
	assert.Equal(t, "a.html", metadata.Links[0].Results[0].TestPath)
	assert.Equal(t, "d.html", metadata.Links[0].Results[1].TestPath)
}
