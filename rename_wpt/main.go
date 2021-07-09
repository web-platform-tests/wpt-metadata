package main

import (
	"context"
	"fmt"
	"path"

	"github.com/google/go-github/v35/github"
	"github.com/web-platform-tests/wpt-metadata/go_util"
)

// This main() gets all the WPT test renaming and deletions between two WPT SHAs, then it deletes/renames
// appropriately in the wpt-metadata repository. This program will roll in the latest WPT SHA once a day
// through a cron job.
func main() {
	// TODO(Kyleju): plumb the before, after SHA through shell. Update SHA through shell as well.
	shaAfter := "e021a4c8f92596f3261e2d7540d017f5138deb68"
	shaBefore := "012651bc39bb45a6dc9c93a968b47dd9310a1e8a"

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
