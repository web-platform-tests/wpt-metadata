package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/web-platform-tests/wpt-metadata/go_util"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

func main() {
	resolveMergeConflict()
}

// When two people try to link different bugs under the same directory at the same time,
// one of two PRs will merge first, and the other PR will likely be stuck due to merge conflict.
// This function will reapply the pending metadata from /api/metadata/pending to the head of
// the wpt-metadat repo.
func resolveMergeConflict() {
	pendingMetadata := getPendingMetadata()
	for testName, links := range pendingMetadata {
		for _, link := range links {
			go_util.WriteMetadataLink(testName[1:], link)
		}
	}
}

// getPendingMetadata fetches pending metadata from
// https://github.com/web-platform-tests/wpt.fyi/blob/main/api/README.md#apimetadatapending.
func getPendingMetadata() shared.MetadataResults {
	log.Println("Fetching pending metadata")
	resp, err := http.Get("https://wpt.fyi/api/metadata/pending")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var metadata shared.MetadataResults
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		log.Fatal(err)
	}

	if len(metadata) == 0 {
		log.Println("No pending metadata")
	}
	return metadata
}
