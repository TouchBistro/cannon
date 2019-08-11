package git

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
)

func Clone(orgName, repoName, destDir string) error {
	repoURL := fmt.Sprintf("git@github.com:%s/%s.git", orgName, repoName)
	destPath := fmt.Sprintf("%s/%s", destDir, repoName)

	_, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Depth:    1, // Try this for now for efficiency
	})
	return errors.Wrapf(err, "failed to clone %s/%s to %s", orgName, repoName, destDir)
}

func PullRequest(repo, branch string) {
	// TODO make this use the github api to actually create the PR
	url := fmt.Sprintf("https://github.com/%s/pull/new/%s", repo, branch)
	fmt.Printf("Pull Request URL for %s is: %s\n", repo, url)
}
