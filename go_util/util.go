// Copyright 2019 The WPT Dashboard Project. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package go_util

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/web-platform-tests/wpt.fyi/shared"
	"gopkg.in/yaml.v3"
)

func appendMetadataLink(testName string, link shared.MetadataLink, metadata *shared.Metadata) {
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
}

// WriteMetadataLink adds a metadataLink to a wpt-metadata META.yml
// entry and writes it to disk.
//
// Directories and META.yml files are created as needed, or an existing one may
// be updated.
func WriteMetadataLink(testName string, link shared.MetadataLink) {
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

	appendMetadataLink(testName, link, &metadata)
	writeToFile(metadata, f)
}

// deleteMetadata deletes testname from metadata and return the new metadata and deleted metadataLink entries.
func deleteMetadata(testname string, metadata shared.Metadata) (shared.Metadata, shared.MetadataLinks) {
	deletedMetadataLinks := shared.MetadataLinks{}
	newMetadataLinks := []shared.MetadataLink{}
	for _, link := range metadata.Links {
		deletedResults := []shared.MetadataTestResult{}
		newResults := []shared.MetadataTestResult{}
		for _, result := range link.Results {
			if result.TestPath == testname {
				// Add the deleted result to the metadataLinks
				deletedResults = append(deletedResults, result)
			} else {
				newResults = append(newResults, result)
			}
		}
		if len(deletedResults) > 0 {
			deletedLink := shared.MetadataLink{Product: link.Product, URL: link.URL, Results: deletedResults}
			deletedMetadataLinks = append(deletedMetadataLinks, deletedLink)
		}

		if len(newResults) > 0 {
			newLink := shared.MetadataLink{Product: link.Product, URL: link.URL, Results: newResults}
			newMetadataLinks = append(newMetadataLinks, newLink)
		}
	}
	newMetadata := shared.Metadata{Links: newMetadataLinks}
	return newMetadata, deletedMetadataLinks
}

// DeleteTestFromMetadata deletes a WPT testPath and returns the deleted metadata entries, if exists.
func DeleteTestFromMetadata(testPath string) shared.MetadataLinks {
	metadata := readMetadataFile(testPath)
	if metadata == nil {
		return shared.MetadataLinks{}
	}

	testname := path.Base(testPath)
	newMetadata, metadataLinks := deleteMetadata(testname, *metadata)
	// Nothing is deleted.
	if len(metadataLinks) == 0 {
		return shared.MetadataLinks{}
	}

	fmt.Println("Removing", testPath, "from the wpt-metadata repo")
	// Everything is deleted
	if len(newMetadata.Links) == 0 {
		removeMetadataFile(testPath)
	} else {
		filepath := getYMLFilePath(testPath)
		f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		writeToFile(newMetadata, f)
	}
	return metadataLinks
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

func readMetadataFile(testPath string) *shared.Metadata {
	filepath := getYMLFilePath(testPath)
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// The file does not exist.
		return nil
	}

	f, err := os.OpenFile(filepath, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	var metadata shared.Metadata
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		log.Fatal(err)
	}
	return &metadata
}

func removeMetadataFile(testPath string) {
	filepath := getYMLFilePath(testPath)
	fmt.Println("Delete", filepath, "from the wpt-metadata repo")
	err := os.Remove(filepath)

	if err != nil {
		log.Fatal(err)
	}
}

func getYMLFilePath(testPath string) string {
	return path.Dir(testPath) + "/META.yml"
}
