package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/google/go-github/v35/github"
	"github.com/web-platform-tests/wpt-metadata/go_util"
)

// This main() gets all the WPT test renaming and deletions between two WPT SHAs, then it deletes/renames
// appropriately in the wpt-metadata repository. This program will roll in the latest WPT SHA once a day
// through a cron job.
func main() {
	shaAfter := updateManifest()
	if shaAfter == "" {
		fmt.Println("Unable to update manifest and get shaAfter")
		return
	}

	data, err := ioutil.ReadFile("go_test/WPT_SHA.json")
	if err != nil {
		fmt.Println("Unable to read go_test/WPT_SHA.json")
		return
	}

	err = ioutil.WriteFile("go_test/WPT_SHA.json", []byte(shaAfter), 0644)
	if err != nil {
		fmt.Println("Unable to update go_test/WPT_SHA.json with shaAfter")
		return
	}
	shaBefore := string(data)

	// Plumb the SHA information from a text file, WPT SHA rolls on a daily basis
	renames, deletes := getChangesBetweenSHAs(context.Background(), shaBefore, shaAfter)
	if renames == nil {
		return
	}

	fmt.Println("Renaming WPT tests...")
	for before, after := range renames {
		renameMetadata(before, after)
	}

	fmt.Println("Deleting WPT tests...")
	for _, delete := range deletes {
		go_util.DeleteTestFromMetadata(delete)
	}
}

// renameMetadata deletes the before WPT test and adds the after WPT test, if exists.
func renameMetadata(before, after string) {
	metadataLinks := go_util.DeleteTestFromMetadata(before)
	if len(metadataLinks) == 0 {
		return
	}

	afterTestname := path.Base(after)
	// Rename the before test name to the after test name.
	for linkIndex, link := range metadataLinks {
		for resultIndex := range link.Results {
			fmt.Println("Renaming", before, after)
			metadataLinks[linkIndex].Results[resultIndex].TestPath = afterTestname
		}
	}

	for _, link := range metadataLinks {
		go_util.WriteMetadataLink(after, link)
	}
}

// getChangesBetweenSHAs returns all the renamed and deleted files between two WPT SHAs.
// TODO(kyleju): In WPT tests, if we have a foo.any.js file, that would have tests
// foo.any.html, foo.any.worker.html and foo.any.sharedworker.html. To perfectly rename/map all tests,
// we need MANIFEST information.
func getChangesBetweenSHAs(ctx context.Context, shaBefore, shaAfter string) (map[string]string, []string) {
	fmt.Println("Getting WPT changes bewtween", shaBefore, "and", shaAfter)
	if shaBefore == shaAfter {
		fmt.Println("Nothing to do since the before/after SHAs are the same:", shaBefore)
		return nil, nil
	}
	githubClient := github.NewClient(nil)
	comparison, _, err := githubClient.Repositories.CompareCommits(ctx, "web-platform-tests", "wpt", shaBefore, shaAfter)
	if err != nil || comparison == nil {
		fmt.Println("Failed to fetch diff for", shaBefore[:7], shaAfter[:7], ":", err.Error())
		return nil, nil
	}

	renames := make(map[string]string)
	deletes := make([]string, 0)
	for _, file := range comparison.Files {
		if file.GetStatus() == "renamed" {
			is, was := file.GetFilename(), file.GetPreviousFilename()
			renames[was] = is
		}

		if file.GetStatus() == "removed" {
			deletes = append(deletes, file.GetFilename())
		}
	}
	return renames, deletes
}

// updateManifest updates go_test/MANIFEST.json and returns the latest WPT SHA.
func updateManifest() string {
	resp, err := http.Get("https://wpt.fyi/api/manifest?sha=latest")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	err = ioutil.WriteFile("go_test/MANIFEST.json", data, 0644)
	if err != nil {
		return ""
	}
	sha := resp.Header.Get("x-wpt-sha")
	return sha
}
