package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
)

func Clone(orgName, repoName, destDir string) (*git.Repository, *git.Worktree, error) {
	repoURL := fmt.Sprintf("git@github.com:%s/%s.git", orgName, repoName)
	destPath := fmt.Sprintf("%s/%s", destDir, repoName)

	r, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
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

func Add(repoName, repoPath, addPath string) error {
	err := util.Exec("git", repoPath, "add", addPath)
	return errors.Wrapf(err, "exec failed to add %s in repo %s", addPath, repoName)
}

func User() (string, string, error) {
	args := []string{"config", "--get", "--global"}
	nameOutput, err := util.ExecOutput("git", append(args, "user.name")...)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get git user name")
	}

	emailOutput, err := util.ExecOutput("git", append(args, "user.email")...)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get git user email")
	}

	return strings.TrimSpace(nameOutput), strings.TrimSpace(emailOutput), nil
}
