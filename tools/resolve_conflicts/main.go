package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/web-platform-tests/wpt.fyi/shared"
	"go.yaml.in/yaml/v4"
)

func main() {
	prNum := flag.String("pr", os.Getenv("PR_NUMBER"), "PR number to resolve conflicts for")
	flag.Parse()

	if *prNum != "" {
		if err := resolvePR(*prNum); err != nil {
			log.Fatalf("Failed to resolve PR #%s: %v", *prNum, err)
		}
		return
	}

	// Local working directory mode: resolve any conflicted META.yml files currently in git index
	if err := resolveLocalIndex(); err != nil {
		log.Fatalf("Failed to resolve local index conflicts: %v", err)
	}
}

func resolvePR(prNum string) error {
	log.Printf("Checking PR #%s for merge conflicts...", prNum)
	branchCmd := exec.Command("gh", "pr", "view", prNum, "--json", "headRefName", "-q", ".headRefName")
	branchOut, err := branchCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get head branch for PR #%s: %w", prNum, err)
	}
	branch := strings.TrimSpace(string(branchOut))
	if branch == "" {
		return fmt.Errorf("empty head branch for PR #%s", prNum)
	}

	runGit("fetch", "origin", branch)
	runGit("checkout", branch)

	// Attempt merge without commit
	err = runGitErr("merge", "origin/master", "--no-commit", "--no-ff")
	if err == nil {
		log.Printf("PR #%s merged origin/master cleanly without conflicts.", prNum)
		runGit("merge", "--abort")
		return nil
	}

	conflictedFiles, err := getConflictedFiles()
	if err != nil || len(conflictedFiles) == 0 {
		runGit("merge", "--abort")
		return fmt.Errorf("merge failed but no conflicted files found: %v", err)
	}

	for _, file := range conflictedFiles {
		if !strings.HasSuffix(strings.ToLower(file), "meta.yml") {
			runGit("merge", "--abort")
			return fmt.Errorf("non-META.yml conflict detected in %s; leaving for manual resolution", file)
		}
		if err := resolveConflictedMetaYML(file); err != nil {
			runGit("merge", "--abort")
			return fmt.Errorf("failed to resolve META.yml conflict in %s: %w", file, err)
		}
	}

	commitMsg := "Auto-resolve META.yml merge conflicts against master"
	if err := runGitErr("commit", "-m", commitMsg); err != nil {
		runGit("merge", "--abort")
		return fmt.Errorf("failed to commit resolved conflicts: %w", err)
	}

	revCmd := exec.Command("git", "rev-parse", "HEAD")
	shaOut, _ := revCmd.Output()
	sha := strings.TrimSpace(string(shaOut))

	log.Printf("Pushing resolved merge commit %s to branch %s...", sha, branch)
	if err := runGitErr("push", "origin", branch); err != nil {
		return fmt.Errorf("failed to push resolved commit: %w", err)
	}

	postRevertComment(prNum, sha, branch)
	return nil
}

func postRevertComment(prNum, sha, branch string) {
	commentBody := fmt.Sprintf(`🤖 **Automated Merge Conflict Resolution**

This PR had Git line conflicts on `+"`META.yml`"+` against `+"`master`"+`. The conflicts were automatically resolved by semantically unioning the metadata links from both branches.

**Commit pushed:** `+"`%s`"+`

### How to Revert or Override
If you need to manually undo this automated resolution, run:
`+"```bash"+`
git fetch origin
git checkout %s
git revert -m 1 %s
git push origin %s
`+"```"+`
Or to manually re-resolve:
`+"```bash"+`
git reset --hard HEAD~1
git merge origin/master
`+"```", sha, branch, sha, branch)

	cmd := exec.Command("gh", "pr", "comment", prNum, "--body", commentBody)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to post PR comment on #%s: %v", prNum, err)
	}
}

func resolveLocalIndex() error {
	conflictedFiles, err := getConflictedFiles()
	if err != nil {
		return err
	}
	if len(conflictedFiles) == 0 {
		log.Println("No conflicted files detected in git index.")
		return nil
	}
	for _, file := range conflictedFiles {
		if !strings.HasSuffix(strings.ToLower(file), "meta.yml") {
			return fmt.Errorf("non-META.yml conflict detected in %s", file)
		}
		if err := resolveConflictedMetaYML(file); err != nil {
			return err
		}
	}
	log.Println("Successfully resolved all META.yml conflicts in local index.")
	return nil
}

func getConflictedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			files = append(files, strings.TrimSpace(l))
		}
	}
	return files, nil
}

func resolveConflictedMetaYML(filePath string) error {
	log.Printf("Resolving Git stages for conflicted file: %s", filePath)
	oursData, err := exec.Command("git", "show", ":2:"+filePath).Output()
	if err != nil {
		return fmt.Errorf("failed to read stage 2 (ours) for %s: %w", filePath, err)
	}
	theirsData, err := exec.Command("git", "show", ":3:"+filePath).Output()
	if err != nil {
		return fmt.Errorf("failed to read stage 3 (theirs) for %s: %w", filePath, err)
	}

	merged, err := semanticUnionMetaYML(oursData, theirsData)
	if err != nil {
		return fmt.Errorf("semantic YAML union failed for %s: %w", filePath, err)
	}

	if err := os.WriteFile(filePath, merged, 0644); err != nil {
		return fmt.Errorf("failed to write resolved file %s: %w", filePath, err)
	}

	return runGitErr("add", filePath)
}

func semanticUnionMetaYML(oursData, theirsData []byte) ([]byte, error) {
	var ours, theirs shared.Metadata
	if len(bytes.TrimSpace(oursData)) > 0 {
		if err := yaml.Unmarshal(oursData, &ours); err != nil {
			return nil, fmt.Errorf("unmarshal stage 2 failed: %w", err)
		}
	}
	if len(bytes.TrimSpace(theirsData)) > 0 {
		if err := yaml.Unmarshal(theirsData, &theirs); err != nil {
			return nil, fmt.Errorf("unmarshal stage 3 failed: %w", err)
		}
	}

	merged := unionMetadata(ours, theirs)
	var buf bytes.Buffer
	d, err := yaml.NewDumper(&buf, yaml.WithCompactSeqIndent(), yaml.WithIndent(2))
	if err != nil {
		return nil, err
	}
	defer d.Close()
	if err := d.Dump(merged); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unionMetadata(ours, theirs shared.Metadata) shared.Metadata {
	for _, tLink := range theirs.Links {
		found := false
		for i, oLink := range ours.Links {
			if oLink.URL == tLink.URL && oLink.Product.MatchesProductSpec(tLink.Product) && oLink.Label == tLink.Label {
				seen := make(map[string]bool)
				for _, r := range oLink.Results {
					key := fmt.Sprintf("%s|%v|%v", r.TestPath, r.SubtestName, r.Status)
					seen[key] = true
				}
				for _, r := range tLink.Results {
					key := fmt.Sprintf("%s|%v|%v", r.TestPath, r.SubtestName, r.Status)
					if !seen[key] {
						ours.Links[i].Results = append(ours.Links[i].Results, r)
						seen[key] = true
					}
				}
				found = true
				break
			}
		}
		if !found {
			ours.Links = append(ours.Links, tLink)
		}
	}
	return ours
}

func runGit(args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func runGitErr(args ...string) error {
	cmd := exec.Command("git", args...)
	return cmd.Run()
}
