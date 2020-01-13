package git

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Repository = git.Repository

func Clone(orgName, repoName, destDir string) (*git.Repository, error) {
	repoURL := fmt.Sprintf("git@github.com:%s/%s.git", orgName, repoName)
	destPath := fmt.Sprintf("%s/%s", destDir, repoName)

	r, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})

	return r, errors.Wrapf(err, "failed to clone %s/%s to %s", orgName, repoName, destDir)
}

func Open(path string) (*git.Repository, error) {
	r, err := git.PlainOpen(path)
	return r, errors.Wrapf(err, "failed to open repo at path %s", path)
}

func CleanAndCheckout(r *git.Repository, branch, name string) error {
	w, err := r.Worktree()
	if err != nil {
		return errors.Wrapf(err, "failed to get worktree for repo %s", name)
	}

	err = w.Clean(&git.CleanOptions{
		Dir: true,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to clean repo %s", name)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Force:  true,
	})

	return errors.Wrapf(err, "failed to checkout %s branch in repo %s", branch, name)
}

func Pull(r *git.Repository, name string) error {
	w, err := r.Worktree()
	if err != nil {
		return errors.Wrapf(err, "failed to get worktree for repo %s", name)
	}

	err = w.Pull(&git.PullOptions{
		SingleBranch: true,
		Progress:     os.Stdout,
	})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return errors.Wrapf(err, "failed to pull changes from remote for repo %s", name)
}

func DeleteBranch(r *git.Repository, branch, name string) error {
	err := r.Storer.RemoveReference(plumbing.NewBranchReferenceName(name))
	return errors.Wrapf(err, "failed to delete branch %s in repo %s", branch, name)
}

func CreateBranch(r *git.Repository, branch, name string) error {
	headRef, err := r.Head()
	if err != nil {
		return errors.Wrapf(err, "failed to get HEAD for repo %s", name)
	}

	branchRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), headRef.Hash())
	w, err := r.Worktree()
	if err != nil {
		return errors.Wrapf(err, "failed to get worktree for repo %s", name)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash:   branchRef.Hash(),
		Branch: branchRef.Name(),
		Create: true,
	})

	return errors.Wrapf(err, "failed to checkout branch %s in repo %s", branch, name)
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

func Commit(r *git.Repository, msg, name string) error {
	userName, email, err := User()
	if err != nil {
		return errors.Wrap(err, "failed to get git user info")
	}

	w, err := r.Worktree()
	if err != nil {
		return errors.Wrapf(err, "failed to get worktree for repo %s", name)
	}

	_, err = w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  userName,
			Email: email,
			When:  time.Now(),
		},
	})

	return errors.Wrapf(err, "failed to commit changes in repo %s", name)
}

func Push(r *git.Repository, name string) error {
	err := r.Push(&git.PushOptions{
		RemoteName: "origin",
	})

	return errors.Wrapf(err, "failedd to push to remote in repo %s", name)
}
