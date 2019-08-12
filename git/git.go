package git

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
)

func Clone(orgName, repoName, destDir string) (*git.Repository, *git.Worktree, error) {
	repoURL := fmt.Sprintf("git@github.com:%s/%s.git", orgName, repoName)
	destPath := fmt.Sprintf("%s/%s", destDir, repoName)

	r, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Depth:    1, // Try this for now for efficiency
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to clone %s/%s to %s", orgName, repoName, destDir)
	}

	w, err := r.Worktree()
	return r, w, errors.Wrapf(err, "failed to get worktree for repo %s/%s", orgName, repoName)
}

func Pull(w *git.Worktree, name string) error {
	err := w.Pull(&git.PullOptions{
		SingleBranch: true,
		Progress:     os.Stdout,
	})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return errors.Wrapf(err, "failed to pull changes from remote for repo %s", name)
}

func PullRequest(repo, branch string) {
	// TODO make this use the github api to actually create the PR
	url := fmt.Sprintf("https://github.com/%s/pull/new/%s", repo, branch)
	fmt.Printf("Pull Request URL for %s is: %s\n", repo, url)
}
