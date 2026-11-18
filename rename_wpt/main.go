package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"

	"github.com/google/go-github/v35/github"
	"github.com/web-platform-tests/wpt-metadata/go_util"
)

const manifestPath = "go_test/MANIFEST.json"

// This main() gets all the WPT test renaming and deletions between two WPT SHAs, then it deletes/renames
// appropriately in the wpt-metadata repository. This program will roll in the latest WPT SHA once a day
// through a cron job.
func main() {
	// Read the current manifest before it is overwritten so that renames can be
	// detected by diffing the old and new manifests.
	oldManifest, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Printf("Unable to read %s; manifest-based rename detection is disabled: %s", manifestPath, err)
	}

	shaAfter := updateManifest()
	if shaAfter == "" {
		log.Fatal("Unable to update manifest and get shaAfter")
	}

	data, err := ioutil.ReadFile("go_test/WPT_SHA.json")
	if err != nil {
		log.Fatalf("Unable to read go_test/WPT_SHA.json: %s", err)
	}

	err = ioutil.WriteFile("go_test/WPT_SHA.json", []byte(shaAfter), 0644)
	if err != nil {
		log.Fatalf("Unable to update go_test/WPT_SHA.json with shaAfter: %s", err)
	}
	shaBefore := string(data)
	if shaBefore == shaAfter {
		fmt.Println("Nothing to do since the before/after SHAs are the same:", shaBefore)
		return
	}

	// Plumb the SHA information from a text file, WPT SHA rolls on a daily basis
	renames, deletes := getChangesBetweenSHAs(context.Background(), shaBefore, shaAfter)
	if renames == nil {
		renames = make(map[string]string)
	}

	// The GitHub compare API returns at most 300 changed files, so large WPT
	// rolls silently miss renames. Supplement it with renames detected by
	// diffing the old and new manifests.
	if newManifest, err := ioutil.ReadFile(manifestPath); err == nil && oldManifest != nil {
		manifestRenames, err := getRenamesFromManifests(oldManifest, newManifest)
		if err != nil {
			fmt.Println("Failed to detect renames from manifests:", err)
		} else {
			added := mergeRenames(renames, manifestRenames)
			fmt.Println("Found", added, "additional renames by diffing manifests")
		}
	}
	deletes = filterDeletes(deletes, renames)

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

// manifestFiles returns a map from file path to content hash for every test
// file listed in a WPT MANIFEST.json.
func manifestFiles(data []byte) (map[string]string, error) {
	var manifest struct {
		Items map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	files := make(map[string]string)
	for testType, trie := range manifest.Items {
		if err := walkManifestTrie(trie, "", files); err != nil {
			return nil, fmt.Errorf("invalid %s manifest items: %w", testType, err)
		}
	}
	return files, nil
}

// walkManifestTrie walks a manifest path trie. Directories are JSON objects;
// files are JSON arrays whose first element is the file's content hash.
func walkManifestTrie(raw json.RawMessage, prefix string, files map[string]string) error {
	var dir map[string]json.RawMessage
	if err := json.Unmarshal(raw, &dir); err == nil {
		for name, child := range dir {
			if err := walkManifestTrie(child, path.Join(prefix, name), files); err != nil {
				return err
			}
		}
		return nil
	}

	var file []json.RawMessage
	if err := json.Unmarshal(raw, &file); err != nil || len(file) == 0 {
		return fmt.Errorf("unexpected manifest entry for %q", prefix)
	}
	var hash string
	// A null hash is left empty and ignored by rename detection.
	_ = json.Unmarshal(file[0], &hash)
	files[prefix] = hash
	return nil
}

// getRenamesFromManifests returns renamed test files between two WPT
// manifests. A file counts as renamed when it disappears from the old manifest
// and exactly one new file has the same content hash.
func getRenamesFromManifests(oldData, newData []byte) (map[string]string, error) {
	oldFiles, err := manifestFiles(oldData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse old manifest: %w", err)
	}
	newFiles, err := manifestFiles(newData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new manifest: %w", err)
	}

	removedByHash := make(map[string][]string)
	for p, hash := range oldFiles {
		if _, ok := newFiles[p]; !ok && hash != "" {
			removedByHash[hash] = append(removedByHash[hash], p)
		}
	}
	addedByHash := make(map[string][]string)
	for p, hash := range newFiles {
		if _, ok := oldFiles[p]; !ok && hash != "" {
			addedByHash[hash] = append(addedByHash[hash], p)
		}
	}

	renames := make(map[string]string)
	for hash, removed := range removedByHash {
		added := addedByHash[hash]
		// Skip ambiguous matches (e.g. identical files).
		if len(removed) == 1 && len(added) == 1 {
			renames[removed[0]] = added[0]
		}
	}
	return renames, nil
}

// mergeRenames adds renames from extra that are not already in renames, and
// returns the number added.
func mergeRenames(renames, extra map[string]string) int {
	added := 0
	for before, after := range extra {
		if _, ok := renames[before]; !ok {
			renames[before] = after
			added++
		}
	}
	return added
}

// filterDeletes removes paths that are handled as renames.
func filterDeletes(deletes []string, renames map[string]string) []string {
	filtered := make([]string, 0, len(deletes))
	for _, d := range deletes {
		if _, ok := renames[d]; !ok {
			filtered = append(filtered, d)
		}
	}
	return filtered
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
	err = ioutil.WriteFile(manifestPath, data, 0644)
	if err != nil {
		return ""
	}
	sha := resp.Header.Get("x-wpt-sha")
	return sha
}
