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
			toWPTMetadata(testName[1:], link)
		}
	}
}

// toWPTMetadata converts a pending metadata entry to a wpt-metadata META.yml
// entry and writes it to disk.
//
// Directories and META.yml files are created as needed, or an existing one may
// be updated.
func toWPTMetadata(testName string, link shared.MetadataLink) {
	dirpath := path.Dir(testName)
	testpath := path.Base(testName)
	createDirIfNeeded(dirpath)

	// Open the META.yml file that we would write to, if it exists. In the case
	// where it does exist we don't want to overwrite the file, so open in
	// read-write-or-create mode.
	filepath := dirpath + "/META.yml"
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0755)
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
			Links: []shared.MetadataLink{
				{
					Product: link.Product,
					URL:     link.URL,
					Results: []shared.MetadataTestResult{{
						TestPath: testpath,
					}},
				},
			},
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

	// First check whether we have an existing crbug link in this META.yml; if
	// so we just want to append our testname to the set of tests associated
	// with it.
	hasAdded := false
	for index, existingLink := range metadata.Links {
		if !link.Product.MatchesProductSpec(existingLink.Product) {
			continue
		}

		// Check whether we are already listed in this entry.
		for _, result := range existingLink.Results {
			if result.TestPath == testpath {
				fmt.Println("EXISTING-LINK: for test", testName, "with url", link.URL)
				return
			}
		}

		if existingLink.URL == link.URL {
			fmt.Println("APPEND-LINK: for test", testName, "with url", link.URL)
			metadata.Links[index].Results = append(existingLink.Results, shared.MetadataTestResult{TestPath: testpath})
			hasAdded = true
			break
		}
	}

	// Otherwise, this is a brand new crbug entry in the META.yml, so add a new
	// link to it.
	if !hasAdded {
		fmt.Println("NEW-LINK: for test", testName, "with url", link.URL)
		newLink := shared.MetadataLink{
			Product: link.Product,
			URL:     link.URL,
			Results: []shared.MetadataTestResult{{
				TestPath: testpath,
			}},
		}
		metadata.Links = append(metadata.Links, newLink)
	}

	writeToFile(metadata, f)
}

// createDirIfNeeded is a helper function to create a directory if it doesn't
// already exist. It handles recursive directory creation.
func createDirIfNeeded(dir string) {
	_, err := os.Stat(dir)
	if !os.IsNotExist(err) {
		return
	}
	os.MkdirAll(dir, os.ModePerm)
}

// writeToFile writes out a wpt.fyi shared.Metadata entry to a given file handle.
//
// TODO: This seems to cause a lot of existing metadata files to be completely
// rewritten; what's the underlying problem here?
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

// getPendingMetadata fetches pending metadata from wpt.fyi/api/metadata/pending.
func getPendingMetadata() shared.MetadataResults {
	log.Println("Fetching pending metadata")
	resp, err := http.Get("https://wpt.fyi/api/metadata/pending")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var metadata shared.MetadataResults
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		panic(err)
	}

	if len(metadata) == 0 {
		log.Println("No pending metadata")
	}
	return metadata
}
