package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/web-platform-tests/wpt.fyi/shared"
	"gopkg.in/yaml.v3"
)

func main() {
	resolveMergeConflict()
}

func resolveMergeConflict() {
	pendingMetadata := getPendingMetadata()
	for testName, links := range pendingMetadata {
		for _, link := range links {
			writePendingMetadata(testName[1:], link)
		}
	}
}

// writePendingMetadata converts a pending metadata entry to a wpt-metadata META.yml
// entry and writes it to disk.
//
// When two people try to link different bugs under the same directory at the same time,
// one of two PRs will merge first, and the other PR will likely be stuck due to merge conflict.
// This function will reapply the pending metadata from /api/metadata/pending to the head of
// the wpt-metadat repo.
//
// Directories and META.yml files are created as needed, or an existing one may
// be updated.
func writePendingMetadata(testName string, link shared.MetadataLink) {
	dirpath := path.Dir(testName)
	os.MkdirAll(dirpath, os.ModePerm)

	// Open the META.yml file that we would write to, if it exists. In the case
	// where it does exist we don't want to overwrite the file, so open in
	// read-write-or-create mode.
	filepath := dirpath + "/META.yml"
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if fi.Size() == 0 {
		// This is an entirely new META.yml file, so create a new
		// shared.Metadata representation for our expectation and write it out.
		fmt.Println("CREATED-META-YML: for test", testName, "with url", link.URL)
		metadata := shared.Metadata{
			Links: []shared.MetadataLink{link},
		}
		writeToFile(metadata, f)
		return
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	var metadata shared.Metadata
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		log.Fatal(err)
	}

	// First check whether we have an existing link in this META.yml; if
	// so we just want to append our testname to the set of tests associated
	// with it.
	hasAdded := false
	for index, existingLink := range metadata.Links {
		if existingLink.URL == link.URL && link.Product.MatchesProductSpec(existingLink.Product) {
			fmt.Println("APPEND-LINK: for test", testName, "with url", link.URL)
			metadata.Links[index].Results = append(existingLink.Results, link.Results...)
			hasAdded = true
			break
		}
	}

	// Otherwise, this is a brand new entry in the META.yml, so add a new
	// link to it.
	if !hasAdded {
		fmt.Println("NEW-LINK: for test", testName, "with url", link.URL)
		metadata.Links = append(metadata.Links, link)
	}

	writeToFile(metadata, f)
}

// writeToFile writes out a wpt.fyi shared.Metadata entry to a given file handle.
//
// TODO(kyleju): https://github.com/web-platform-tests/wpt.fyi/issues/1957.
func writeToFile(metadata shared.Metadata, f *os.File) {
	metadataBytes, err := yaml.Marshal(metadata)
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteAt(metadataBytes, 0)
	if err != nil {
		log.Fatal(err)
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
