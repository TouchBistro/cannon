package git

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/progress"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository holds the state of a Git repository.
type Repository struct {
	name       string
	path       string
	baseBranch string
	r          *git.Repository
	w          *git.Worktree
}

// Name returns the name of the repository.
func (repo *Repository) Name() string {
	return repo.name
}

// Path returns the path of the cloned repository on the OS filesystem.
func (repo *Repository) Path() string {
	return repo.path
}

// Prepare prepares the repo for use and returns a Repository instance.
// If the repo does not exist, it will be cloned to dir. Otherwise, any
// uncommitted changes will be discarded and the base branch will be updated.
func Prepare(ctx context.Context, name, dir, baseBranch string) (*Repository, error) {
	tracker := progress.TrackerFromContext(ctx)
	path := filepath.Join(dir, name)
	repo := &Repository{name: name, path: path, baseBranch: baseBranch}
	skipCleanup := false
	var err error
	if !file.Exists(path) {
		// If the repo doesn't exist all we need to do is clone it.
		// Don't need to worry about any dirty state.
		skipCleanup = true
		tracker.Debugf("Repo %s does not exist, cloning", name)
		repo.r, err = git.PlainCloneContext(ctx, path, false, &git.CloneOptions{
			URL: fmt.Sprintf("git@github.com:%s.git", name),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to clone %s to %s: %w", name, dir, err)
		}
		tracker.Debugf("Cloned repo %s to %s", name, dir)
	} else {
		repo.r, err = git.PlainOpen(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open repo at path %s: %w", path, err)
		}
	}

	// Get worktree now and save it since most operations require it.
	repo.w, err = repo.r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree for repo %s: %w", name, err)
	}
	if skipCleanup {
		return repo, nil
	}

	// Cannon leaves this things in a dirty state after a run.
	// It is easier to always assume cannon is in a dirty state and clean up, rather
	// than try to ensure cannon cleans up before it exists.
	// This way if the repo is in a dirty state, no matter the reason,
	// cannon can always get it into a clean state.

	// First, clean the current branch by discarding any working state.
	tracker.Debugf("Cleaning and updating repo %s", name)
	err = repo.w.Clean(&git.CleanOptions{Dir: true})
	if err != nil {
		return nil, fmt.Errorf("failed to clean repo %s: %w", name, err)
	}

	// See if we need to switch branches.
	branchRef, err := repo.r.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD of repo %s: %w", name, err)
	}
	if branchRef.Name().Short() != baseBranch {
		err := repo.w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(baseBranch),
			Force:  true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to checkout %s branch in repo %s: %w", baseBranch, name, err)
		}
		// Delete the old branch we were on.
		err = repo.r.Storer.RemoveReference(branchRef.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to delete branch %s in repo %s: %w", branchRef.Name().Short(), name, err)
		}
	}

	// Update branch.
	err = repo.w.PullContext(ctx, &git.PullOptions{SingleBranch: true})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("failed to pull changes from remote for repo %s: %w", name, err)
	}
	tracker.Debugf("Updated repo %s", name)
	return repo, nil
}

// CreateBranch creates a new branch and switches to it.
func (repo *Repository) CreateBranch(branch string) error {
	headRef, err := repo.r.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD for repo %s: %w", repo.name, err)
	}
	branchRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), headRef.Hash())
	err = repo.w.Checkout(&git.CheckoutOptions{
		Hash:   branchRef.Hash(),
		Branch: branchRef.Name(),
		Create: true,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s in repo %s: %w", branch, repo.name, err)
	}
	return nil
}

// CommitChanges will stage all changes and commit them.
func (repo *Repository) CommitChanges(ctx context.Context, msg string) error {
	// Shell out to git add since there were issues trying to do it will the git module.
	var stderr bytes.Buffer
	cmd := command.New(command.WithDir(repo.path), command.WithStderr(&stderr))
	if err := cmd.Exec(ctx, "git", "add", "."); err != nil {
		return fmt.Errorf("failed to stage changes: %s: %w", stderr.String(), err)
	}

	username, email, err := user(ctx)
	if err != nil {
		return err
	}
	_, err = repo.w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  username,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes in repo %s: %w", repo.name, err)
	}
	return nil
}

func (repo *Repository) Push(ctx context.Context) error {
	err := repo.r.PushContext(ctx, &git.PushOptions{RemoteName: "origin"})
	if err != nil {
		return fmt.Errorf("failed to push to remote in repo %s: %w", repo.name, err)
	}
	return nil
}

// user gets the current configured git user.
func user(ctx context.Context) (username, email string, err error) {
	var outbuf, errbuf bytes.Buffer
	cmd := command.New(command.WithStdout(&outbuf), command.WithStderr(&errbuf))
	err = cmd.Exec(ctx, "git", "config", "--get", "--global", "user.name")
	if err != nil {
		err = fmt.Errorf("failed to get git user name: %s: %w", errbuf.String(), err)
		return
	}
	username = strings.TrimSpace(outbuf.String())

	outbuf.Reset()
	errbuf.Reset()
	err = cmd.Exec(ctx, "git", "config", "--get", "--global", "user.email")
	if err != nil {
		err = fmt.Errorf("failed to get git user email: %s: %w", errbuf.String(), err)
		return
	}
	email = strings.TrimSpace(outbuf.String())
	return
}

// GitHub support

func CreatePRURL(repo, branch string) string {
	return fmt.Sprintf("https://github.com/%s/pull/new/%s", repo, branch)
}

func (repo *Repository) CreatePR(ctx context.Context, branch, desc string) (string, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(map[string]string{
		"title": branch,
		"head":  branch,
		"base":  repo.baseBranch,
		"body":  desc,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create JSON body for request: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls", repo.name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create POST request to GitHub API: %w", err)
	}
	token := fmt.Sprintf("token %s", os.Getenv("GITHUB_TOKEN"))
	req.Header.Add("Authorization", token)
	// Use v3 API
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to create PR for repo %s: %w", repo.name, err)
	}
	defer res.Body.Close()
	if res.StatusCode != 201 {
		return "", fmt.Errorf("got %d response from GitHub API", res.StatusCode)
	}

	var rb struct {
		HTMLURL string `json:"html_url"`
	}
	err = json.NewDecoder(res.Body).Decode(&rb)
	if err != nil {
		return "", fmt.Errorf("failed to decode JSON from reponse body: %w", err)
	}
	return rb.HTMLURL, nil
}
