package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const oldManifestJSON = `{
  "version": 8,
  "items": {
    "testharness": {
      "css": {
        "css-backgrounds": {
          "parsing": {
            "border-image-valid.html": ["aaa", [null, {}]],
            "background-valid.html": ["bbb", [null, {}]]
          }
        }
      },
      "dup": {
        "one.html": ["same", [null, {}]],
        "two.html": ["same", [null, {}]]
      },
      "gone": {
        "deleted.html": ["ddd", [null, {}]]
      }
    },
    "reftest": {
      "grid": {
        "row-001.html": ["eee", [null, [["/grid/row-001-ref.html", "=="]], {}]]
      }
    }
  }
}`

const newManifestJSON = `{
  "version": 8,
  "items": {
    "testharness": {
      "css": {
        "css-backgrounds": {
          "parsing": {
            "background-valid.html": ["bbb", [null, {}]]
          }
        },
        "css-borders": {
          "parsing": {
            "border-image-valid.html": ["aaa", [null, {}]]
          }
        }
      },
      "dup2": {
        "one.html": ["same", [null, {}]],
        "two.html": ["same", [null, {}]]
      }
    },
    "reftest": {
      "grid": {
        "row-001.tentative.html": ["eee", [null, [["/grid/row-001-ref.html", "=="]], {}]]
      }
    }
  }
}`

func TestManifestFiles(t *testing.T) {
	files, err := manifestFiles([]byte(oldManifestJSON))
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{
		"css/css-backgrounds/parsing/border-image-valid.html": "aaa",
		"css/css-backgrounds/parsing/background-valid.html":   "bbb",
		"dup/one.html":      "same",
		"dup/two.html":      "same",
		"gone/deleted.html": "ddd",
		"grid/row-001.html": "eee",
	}, files)
}

func TestManifestFiles_Invalid(t *testing.T) {
	_, err := manifestFiles([]byte(`not json`))
	assert.NotNil(t, err)

	_, err = manifestFiles([]byte(`{"items": {"testharness": {"a.html": 5}}}`))
	assert.NotNil(t, err)
}

func TestGetRenamesFromManifests(t *testing.T) {
	renames, err := getRenamesFromManifests([]byte(oldManifestJSON), []byte(newManifestJSON))
	assert.Nil(t, err)
	// Moved and renamed files are detected; ambiguous (identical-content) and
	// deleted files are not.
	assert.Equal(t, map[string]string{
		"css/css-backgrounds/parsing/border-image-valid.html": "css/css-borders/parsing/border-image-valid.html",
		"grid/row-001.html": "grid/row-001.tentative.html",
	}, renames)
}

func TestGetRenamesFromManifests_Invalid(t *testing.T) {
	_, err := getRenamesFromManifests([]byte(`bad`), []byte(newManifestJSON))
	assert.NotNil(t, err)
	_, err = getRenamesFromManifests([]byte(oldManifestJSON), []byte(`bad`))
	assert.NotNil(t, err)
}

func TestMergeRenames(t *testing.T) {
	renames := map[string]string{"a.html": "b.html"}
	added := mergeRenames(renames, map[string]string{
		"a.html": "other.html", // compare API result wins
		"c.html": "d.html",
	})
	assert.Equal(t, 1, added)
	assert.Equal(t, map[string]string{"a.html": "b.html", "c.html": "d.html"}, renames)
}

func TestFilterDeletes(t *testing.T) {
	deletes := filterDeletes([]string{"a.html", "gone.html"}, map[string]string{"a.html": "b.html"})
	assert.Equal(t, []string{"gone.html"}, deletes)
}
