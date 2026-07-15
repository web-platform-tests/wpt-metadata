package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v35/github"
	"github.com/web-platform-tests/wpt-metadata/go_util"
	"github.com/web-platform-tests/wpt.fyi/shared"
	"go.yaml.in/yaml/v4"
)

type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// This main() gets all the WPT test renaming and deletions between two WPT SHAs, then it deletes/renames
// appropriately in the wpt-metadata repository. This program will roll in the latest WPT SHA once a day
// through a cron job.
func main() {
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

	// Plumb the SHA information from a text file, WPT SHA rolls on a daily basis
	renames, deletes := getChangesBetweenSHAs(context.Background(), shaBefore, shaAfter)
	if renames != nil {
		log.Println("Renaming WPT tests...")
		for before, after := range renames {
			renameMetadata(before, after)
		}

		log.Println("Deleting WPT tests...")
		for _, delete := range deletes {
			go_util.DeleteTestFromMetadata(delete)
		}
	}

	if manifestData, err := ioutil.ReadFile("go_test/MANIFEST.json"); err == nil {
		var newManifest shared.Manifest
		if err := json.Unmarshal(manifestData, &newManifest); err == nil {
			sweepStaleMetadataAgainstManifest(&newManifest)
		}
	}
}

func sweepStaleMetadataAgainstManifest(manifest *shared.Manifest) {
	log.Println("Sweeping repository for stale metadata against updated WPT manifest...")
	filepath.Walk(".", func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.ToLower(info.Name()) != "meta.yml" {
			return nil
		}
		fileDir := path.Dir(filePath)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil
		}
		var metadata shared.Metadata
		if err := yaml.Unmarshal(data, &metadata); err != nil {
			return nil
		}
		for _, link := range metadata.Links {
			for _, result := range link.Results {
				if result.TestPath == "" {
					continue
				}
				fullPath := path.Join(fileDir, result.TestPath)
				var ok bool
				if result.TestPath == "*" {
					ok, _ = manifest.ContainsFile(fileDir)
				} else {
					ok, _ = manifest.ContainsTest(fullPath)
				}
				if !ok {
					log.Printf("Pruning stale test path not found in latest manifest: %s", fullPath)
					go_util.DeleteTestFromMetadata(fullPath)
				}
			}
		}
		return nil
	})
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
			log.Printf("Renaming %s to %s", before, after)
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
	log.Printf("Getting WPT changes between %s and %s", shaBefore, shaAfter)
	if shaBefore == shaAfter {
		log.Printf("Nothing to do since the before/after SHAs are the same: %s", shaBefore)
		return nil, nil
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		token = os.Getenv("WPTMETADATA_BOT_USER_PAT")
	}
	var tc *http.Client
	if token != "" {
		tc = &http.Client{Transport: &authTransport{token: token}}
	}
	githubClient := github.NewClient(tc)
	comparison, _, err := githubClient.Repositories.CompareCommits(ctx, "web-platform-tests", "wpt", shaBefore, shaAfter)
	if err != nil || comparison == nil {
		log.Printf("Failed to fetch diff for %s %s: %v", shaBefore[:7], shaAfter[:7], err)
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
