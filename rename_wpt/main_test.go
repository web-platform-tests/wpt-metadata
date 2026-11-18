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
        "deleted.html": ["ddd", [null, {}]],
        "variants.html": ["vvv", ["gone/variants.html?a", {}], ["gone/variants.html?b", {}]],
        "thing.any.js": ["jjj", ["gone/thing.any.html", {}], ["gone/thing.any.worker.html", {}]]
      }
    },
    "reftest": {
      "grid": {
        "row-001.html": ["eee", [null, [["/grid/row-001-ref.html", "=="]], {}]]
      }
    },
    "support": {
      "grid": {
        "row-001-ref.html": ["fff", null]
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
	hashes := make(map[string]string)
	for p, f := range files {
		hashes[p] = f.Hash
	}
	assert.Equal(t, map[string]string{
		"css/css-backgrounds/parsing/border-image-valid.html": "aaa",
		"css/css-backgrounds/parsing/background-valid.html":   "bbb",
		"dup/one.html":       "same",
		"dup/two.html":       "same",
		"gone/deleted.html":  "ddd",
		"gone/variants.html": "vvv",
		"gone/thing.any.js":  "jjj",
		"grid/row-001.html":  "eee",
	}, hashes)
	assert.Equal(t, []string{"gone/deleted.html"}, files["gone/deleted.html"].TestIDs)
	assert.Equal(t, []string{"gone/variants.html?a", "gone/variants.html?b"}, files["gone/variants.html"].TestIDs)
	assert.Equal(t, []string{"gone/thing.any.html", "gone/thing.any.worker.html"}, files["gone/thing.any.js"].TestIDs)
}

func TestManifestFiles_Invalid(t *testing.T) {
	_, err := manifestFiles([]byte(`not json`))
	assert.NotNil(t, err)

	_, err = manifestFiles([]byte(`{"items": {"testharness": {"a.html": 5}}}`))
	assert.NotNil(t, err)
}

func TestDiffManifests(t *testing.T) {
	// The tiny test manifests delete far more than 2% of their files.
	defer func(orig float64) { maxManifestDeleteFraction = orig }(maxManifestDeleteFraction)
	maxManifestDeleteFraction = 1

	diff, err := diffManifests([]byte(oldManifestJSON), []byte(newManifestJSON))
	assert.Nil(t, err)
	// Moved and renamed files are detected; ambiguous (identical-content)
	// files are neither renamed nor deleted.
	assert.Equal(t, map[string]string{
		"css/css-backgrounds/parsing/border-image-valid.html": "css/css-borders/parsing/border-image-valid.html",
		"grid/row-001.html": "grid/row-001.tentative.html",
	}, diff.Renames)
	// Removed files without a successor are deleted, including all of
	// their test IDs (variants, .any.js generated tests).
	assert.Equal(t, []string{
		"gone/deleted.html",
		"gone/thing.any.js",
		"gone/thing.any.html",
		"gone/thing.any.worker.html",
		"gone/variants.html",
		"gone/variants.html?a",
		"gone/variants.html?b",
	}, diff.Deletes)
}

func TestDiffManifests_KeepsTestIDsThatStillExist(t *testing.T) {
	defer func(orig float64) { maxManifestDeleteFraction = orig }(maxManifestDeleteFraction)
	maxManifestDeleteFraction = 1

	oldM := `{"items": {"testharness": {"a": {"foo.any.js": ["h1", ["a/foo.any.html", {}], ["a/foo.any.worker.html", {}]]}}}}`
	// Contrived: an unrelated new file that generates one of the old test IDs.
	newM := `{"items": {"testharness": {"b": {"generator.html": ["h2", ["a/foo.any.html", {}]]}}}}`
	diff, err := diffManifests([]byte(oldM), []byte(newM))
	assert.Nil(t, err)
	assert.Equal(t, []string{"a/foo.any.js", "a/foo.any.worker.html"}, diff.Deletes)
}

func TestDiffManifests_DeletionSafetyLimit(t *testing.T) {
	// Losing most of the manifest looks like a broken download; nothing
	// should be deleted.
	diff, err := diffManifests([]byte(oldManifestJSON), []byte(`{"version": 8, "items": {}}`))
	assert.Nil(t, err)
	assert.Empty(t, diff.Deletes)
	assert.Empty(t, diff.Renames)
}

func TestDiffManifests_Invalid(t *testing.T) {
	_, err := diffManifests([]byte(`bad`), []byte(newManifestJSON))
	assert.NotNil(t, err)
	_, err = diffManifests([]byte(oldManifestJSON), []byte(`bad`))
	assert.NotNil(t, err)
}

func TestMergeDeletes(t *testing.T) {
	assert.Equal(t, []string{"a.html", "b.html", "c.html"},
		mergeDeletes([]string{"a.html", "b.html"}, []string{"b.html", "c.html"}))
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

func TestMapTestIDs(t *testing.T) {
	assert.Equal(t, map[string]string{
		"a/x.html?v=1": "a/x.tentative.html?v=1",
		"a/x.html?v=2": "a/x.tentative.html?v=2",
	}, mapTestIDs("a/x.html", "a/x.tentative.html",
		[]string{"a/x.html?v=1", "a/x.html?v=2", "a/x.html?gone"},
		[]string{"a/x.tentative.html?v=1", "a/x.tentative.html?v=2"}))

	assert.Equal(t, map[string]string{
		"a/foo.tentative.any.html":        "b/foo.any.html",
		"a/foo.tentative.any.worker.html": "b/foo.any.worker.html",
	}, mapTestIDs("a/foo.tentative.any.js", "b/foo.any.js",
		[]string{"a/foo.tentative.any.html", "a/foo.tentative.any.worker.html"},
		[]string{"b/foo.any.html", "b/foo.any.worker.html"}))
}

func TestNormalizeTestName(t *testing.T) {
	assert.Equal(t, "textstream", normalizeTestName("fetch/api/body/textstream.tentative.any.js"))
	assert.Equal(t, "idlharness", normalizeTestName("media-source/idlharness.window.js"))
	assert.Equal(t, "border-image-source-computed", normalizeTestName("a/border-image-source-computed.sub.html"))
	assert.Equal(t, "2d.color.space.p3.to.p3", normalizeTestName("a/2d.color.space.p3.to.p3.html"))
}

func TestNameSimilarity(t *testing.T) {
	assert.Equal(t, 1.0, nameSimilarity("abc", "abc"))
	assert.InDelta(t, 0.87, nameSimilarity("flow-tolerance-computed", "fit-tolerance-computed"), 0.01)
	assert.Less(t, nameSimilarity("clip-path-animation-missing-0-percent", "clip-path-animation-neutral-keyframe"), minRenameSimilarity)
}

func TestDiffManifests_PossibleRenamesAreNotDeleted(t *testing.T) {
	defer func(orig float64) { maxManifestDeleteFraction = orig }(maxManifestDeleteFraction)
	maxManifestDeleteFraction = 1

	oldM := `{"items": {"testharness": {
		"a": {
			"flow-tolerance-computed.html": ["h1", [null, {}]],
			"textstream.tentative.any.js": ["h2", ["a/textstream.tentative.any.html", {}]],
			"really-gone.html": ["h3", [null, {}]]
		},
		"b": {"moved-and-edited.html": ["h4", [null, {}]]}
	}}}`
	newM := `{"items": {"testharness": {
		"a": {
			"fit-tolerance-computed.html": ["h1-edited", [null, {}]],
			"textstream.any.js": ["h2-edited", ["a/textstream.any.html", {}]],
			"unrelated-new-test.html": ["h5", [null, {}]]
		},
		"c": {"moved-and-edited.html": ["h4-edited", [null, {}]]}
	}}}`
	diff, err := diffManifests([]byte(oldM), []byte(newM))
	assert.Nil(t, err)
	assert.Empty(t, diff.Renames)
	assert.Equal(t, []string{"a/really-gone.html"}, diff.Deletes)
	assert.Equal(t, []string{
		"a/flow-tolerance-computed.html -> a/fit-tolerance-computed.html",
		"a/textstream.tentative.any.js -> a/textstream.any.js",
		"b/moved-and-edited.html -> c/moved-and-edited.html",
	}, diff.PossibleRenames)
}
