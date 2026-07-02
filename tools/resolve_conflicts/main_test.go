package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func TestSemanticUnionMetaYML(t *testing.T) {
	oursYAML := []byte(`links:
  - product: chrome
    url: https://crbug.com/12345
    results:
    - test: a.html
      status: FAIL
`)
	theirsYAML := []byte(`links:
  - product: chrome
    url: https://crbug.com/12345
    results:
    - test: b.html
      status: FAIL
  - product: firefox
    url: https://bugzilla.mozilla.org/show_bug.cgi?id=67890
    results:
    - test: c.html
      status: TIMEOUT
`)

	mergedBytes, err := semanticUnionMetaYML(oursYAML, theirsYAML)
	assert.NoError(t, err)

	var merged shared.Metadata
	err = yaml.Unmarshal(mergedBytes, &merged)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(merged.Links))
	assert.Equal(t, "chrome", merged.Links[0].Product.BrowserName)
	assert.Equal(t, 2, len(merged.Links[0].Results))
	assert.Equal(t, "a.html", merged.Links[0].Results[0].TestPath)
	assert.Equal(t, "b.html", merged.Links[0].Results[1].TestPath)

	assert.Equal(t, "firefox", merged.Links[1].Product.BrowserName)
	assert.Equal(t, 1, len(merged.Links[1].Results))
	assert.Equal(t, "c.html", merged.Links[1].Results[0].TestPath)
}

func TestSemanticUnionMetaYML_Deduplication(t *testing.T) {
	oursYAML := []byte(`links:
  - product: chrome
    url: https://crbug.com/12345
    results:
    - test: a.html
      status: FAIL
`)
	theirsYAML := []byte(`links:
  - product: chrome
    url: https://crbug.com/12345
    results:
    - test: a.html
      status: FAIL
    - test: b.html
      status: FAIL
`)

	mergedBytes, err := semanticUnionMetaYML(oursYAML, theirsYAML)
	assert.NoError(t, err)
	mergedStr := string(mergedBytes)

	assert.Equal(t, 1, strings.Count(mergedStr, "a.html"), "a.html should be deduplicated")
	assert.Contains(t, mergedStr, "b.html")
}
