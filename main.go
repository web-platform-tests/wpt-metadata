package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

// expectation represents a parsed TestExpectations entry.
type expectation struct {
	url      string
	tags     string
	testname string
	results  string
	// Used for debugging only
	raw string
}

func main() {
	filterPtr := flag.String("filter", "", "Filter results to only this directory")
	flag.Parse()

	portTestExpectations(*filterPtr)
}

// portTestExpectations parses the Chromium TestExpectations file and converts
// it into wpt-metadata entries where appropriate.
func portTestExpectations(filter string) {
	f, err := os.Open("TestExpectations.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Println("Parsing TestExpectations file")
	var expectations []expectation
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		exp := parseExpectation(line)
		if exp == nil {
			continue
		}

		if meetsCriteria(exp, filter) {
			expectations = append(expectations, *exp)
		}
	}

	fmt.Println("\nConverting to wpt-metadata entries")
	for _, exp := range(expectations) {
		toWPTMetadata(exp)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nDone!")
}

// parseExpectation converts a raw TestExpectation line into a parsed
// expectation struct. If the line is not a valid expectation (e.g. a comment,
// test is not in external/wpt, etc), nil is returned instead.
func parseExpectation(line string) *expectation {
	if strings.HasPrefix(line, "# ") {
		return nil
	}

	// syntax: https://chromium.googlesource.com/chromium/src/+/master/docs/testing/web_test_expectations.md#syntax
	// [ bugs ] [ "[" modifiers "]" ] test_name [ "[" expectations "]" ]
	exp := expectation{
		raw: line,
	}
	parts := strings.Split(line, " ")
	for i := 0; i < len(parts); i++ {
		// Assumption: no test_name starts with crbug.com.
		if strings.HasPrefix(parts[i], "crbug.com") {
			exp.url = parts[i]
			continue
		}

		// This could either be tags (before test_name) or expectations (after).
		if parts[i] == "[" {
			val := ""
			i++
			for parts[i] != "]" {
				val += strings.ToLower(parts[i])
				i++
			}

			if exp.testname == "" {
				exp.tags = val
			} else {
				exp.results = val
			}
			continue
		}

		// This can only be the testname; if its not external/wpt/ we don't care.
		// TODO: Arguably should be in meetsCriteria instead.
		if !strings.HasPrefix(parts[i], "external/wpt/") {
			return nil
		}
		exp.testname = parts[i][len("external/wpt/"):]
	}

	return &exp
}

// meetsCriteria determines if a TestExpectations entry meets our criteria for
// auto-importing to wpt-metadata.
//
// In short, it must:
//   * Have a crbug link,
//   * Have a single, known failure status (e.g. Timeout)
//   * Have happened on a supported platform (i.e. Linux).
//
// See: https://github.com/web-platform-tests/wpt.fyi/issues/1993#issuecomment-654551961
func meetsCriteria(exp *expectation, filter string) bool {
	if exp.testname == "" {
		log.Fatal("Empty testname in meetsCriteria. Raw expectation:", exp.raw)
		return false
	}

	if filter != "" && !strings.HasPrefix(exp.testname, filter) {
		fmt.Println("SKIPPED:", exp.testname, "[ does not match filter:", filter, "]")
		return false
	}

	if exp.url == "" {
		fmt.Println("SKIPPED:", exp.testname, "[ No associated crbug ]")
		return false
	}

	if exp.results != "timeout" && exp.results != "failure" && exp.results != "skip" {
		fmt.Println("SKIPPED:", exp.testname, "[ Unsupported results:", exp.results, "]")
		return false
	}

	if exp.tags != "" {
		if !strings.Contains(exp.tags, "linux") {
			if exp.tags != "release" {
				fmt.Println("SKIPPED:", exp.testname, "[ Unsupported tags:", exp.tags, "]")
				return false
			}
		} else {
			if strings.Contains(exp.tags, "debug") {
				fmt.Println("SKIPPED:", exp.testname, "[ Unsupported tags:", exp.tags, "]")
				return false
			}
		}
	}
	return true
}

// toWPTMetadata converts a TestExpectation entry to a wpt-metadata META.yml
// entry and writes it to disk.
//
// Directories and META.yml files are created as needed, or an existing one may
// be updated.
func toWPTMetadata(exp expectation) {
	dirpath := path.Dir(exp.testname)
	testpath := path.Base(exp.testname)
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
		fmt.Println("CREATED-META-YML: for test", exp.testname, "with url", exp.url)
		metadata := shared.Metadata{
			Links: []shared.MetadataLink{
				{
					Product: shared.ParseProductSpecUnsafe("chrome"),
					URL:     exp.url,
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
	for index, link := range metadata.Links {
		if link.Product.String() != "chrome" {
			continue
		}

		// Check whether we are already listed in this entry.
		for _, result := range link.Results {
			if result.TestPath == testpath {
				fmt.Println("EXISTING-LINK: for test", exp.testname, "with url", exp.url)
				return
			}
		}

		if link.URL == exp.url {
			fmt.Println("APPEND-LINK: for test", exp.testname, "with url", exp.url)
			metadata.Links[index].Results = append(link.Results, shared.MetadataTestResult{TestPath: testpath})
			hasAdded = true
			break
		}
	}

	// Otherwise, this is a brand new crbug entry in the META.yml, so add a new
	// link to it.
	//
	// TODO: This does not handle the case where a single test may be
	// associated with multiple links - should it?
	if !hasAdded {
		fmt.Println("NEW-LINK: for test", exp.testname, "with url", exp.url)
		newLink := shared.MetadataLink{
			Product: shared.ParseProductSpecUnsafe("chrome"),
			URL:     exp.url,
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
