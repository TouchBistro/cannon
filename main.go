package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/config"
	"github.com/TouchBistro/cannon/fatal"
	g "github.com/TouchBistro/cannon/git"
	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	flag "github.com/spf13/pflag"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const (
	greenColor = "\x1b[32m"
	cyanColor  = "\x1b[36m"
	resetColor = "\x1b[0m"
)

var (
	configPath    string
	commitMessage string
)

// Make sure repo is on master with latest changes
func prepareRepo(repo string) (*git.Repository, *git.Worktree, error) {
	parts := strings.Split(repo, "/")
	orgName := parts[0]
	repoName := parts[1]
	path := fmt.Sprintf("%s/%s", config.CannonDir(), repoName)

	// Repo doesn't exist, clone and then we are good to go
	if !util.FileOrDirExists(path) {
		fmt.Printf("Repo %s does not exist, cloning...", repo)

		r, w, err := g.Clone(orgName, repoName, config.CannonDir())
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to clone repo %s", repo)
		}

		fmt.Printf("Cloned repo %s to %s\n", repo, config.CannonDir())
		return r, w, nil
	}

	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to open repo at path %s", path)
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get worktree for repo %s", repo)
	}

	branchRef, err := r.Head()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get HEAD for repo %s", repo)
	}

	// Discard any changes and switch to master
	err = w.Clean(&git.CleanOptions{
		Dir: true,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to clean repo %s", repo)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Force: true,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to switch to master branch in repo %s", repo)
	}

	// Pull latest changes
	err = g.Pull(w, repo)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to update master branch for repo %s", repo)
	}

	// Head returns refs/heads/<BRANCH>, need to get branch
	refParts := strings.Split(string(branchRef.Name()), "/")
	branch := refParts[len(refParts)-1]
	if branch == "master" {
		return r, w, nil
	}

	// Delete old branch
	err = r.Storer.RemoveReference(branchRef.Name())
	return r, w, errors.Wrapf(err, "failed to delete branch %s in repo %s", branch, repo)
}

func executeAction(a config.Action, repoPath string) error {
	switch a.Type {
	case config.ActionReplaceLine:
		return action.ReplaceLine(a, repoPath)
	case config.ActionReplaceText:
		return action.ReplaceText(a, repoPath)
	case config.ActionCreateFile:
		return action.CreateFile(a, repoPath)
	case config.ActionDeleteFile:
		return action.DeleteFile(a, repoPath)
	case config.ActionReplaceFile:
		return action.ReplaceFile(a, repoPath)
	case config.ActionCreateOrReplaceFile:
		return action.CreateOrReplaceFile(a, repoPath)
	case config.ActionRunCommand:
		return action.RunCommand(a, repoPath)
	default:
		return errors.New(fmt.Sprintf("invalid action type %s", a.Type))
	}
}

func performActions(r *git.Repository, w *git.Worktree, actions []config.Action, branchName, repo string) error {
	headRef, err := r.Head()
	if err != nil {
		return errors.Wrapf(err, "failed to get HEAD for repo %s", repo)
	}

	// Create and checkout new branch
	branchRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+branchName), headRef.Hash())
	err = w.Checkout(&git.CheckoutOptions{
		Hash:   branchRef.Hash(),
		Branch: branchRef.Name(),
		Create: true,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create new branch %s in repo %s", branchName, repo)
	}

	// Execute actions
	repoName := strings.Split(repo, "/")[1]
	path := fmt.Sprintf("%s/%s", config.CannonDir(), repoName)
	for _, a := range actions {
		err = executeAction(a, path)
		if err != nil {
			return errors.Wrapf(err, "failed to execute action %s in repo %s", a.Type, repo)
		}
	}

	// Commit changes and push
	err = g.Add(repoName, path, ".")
	if err != nil {
		return errors.Wrapf(err, "failed to stage change files in repo %s", repo)
	}

	name, email, err := g.User()
	if err != nil {
		return errors.Wrap(err, "failed to get git user info")
	}

	_, err = w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes in repo %s", repo)
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
	})
	return errors.Wrapf(err, "failed to push changes to remote for repo %s", repo)
}

func parseFlags() {
	flag.StringVarP(&configPath, "path", "p", "cannon.yml", "The path to a cannon.yml config file")
	flag.StringVarP(&commitMessage, "commit-message", "m", "", "The commit message to use")

	flag.Parse()

	if commitMessage == "" {
		log.Fatalln("Must provide a commit message")
	}
}

func main() {
	parseFlags()

	err := config.Init(configPath)
	if err != nil {
		fatal.ExitErr(err, "Failed reading config file.")
	}

	conf := config.Config()
	fmt.Println("Affected repos:")
	for _, repo := range conf.Repos {
		fmt.Printf("- %s\n", repo)
	}

	// Have user confirm changes
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nConfirm running with these parameters (y/n): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		fatal.ExitErr(err, "Failed to read user input.")
	}

	choice := strings.TrimSpace(input)
	if choice != "y" {
		fmt.Println("Aborting")
		os.Exit(0)
	}
	fmt.Println()

	branchName := "cannon/change-" + uuid.NewV4().String()[0:8]

	// Make sure repos are up to date
	for _, repo := range conf.Repos {
		r, w, err := prepareRepo(repo)

		fmt.Printf("%sRunning actions for repo %s%s\n", cyanColor, repo, resetColor)
		if err != nil {
			fatal.ExitErrf(err, "Failed to prepare repo %s.", repo)
		}

		err = performActions(r, w, conf.Actions, branchName, repo)
		if err != nil {
			fatal.ExitErrf(err, "Failed to perform actions on repo %s.", repo)
		}

		fmt.Printf("%sSuccessfully performed actions for repo %s%s\n\n", greenColor, repo, resetColor)
	}

	fmt.Println("Pull Request URLs:")
	for _, repo := range conf.Repos {
		fmt.Printf("- %s: %s\n", repo, g.PullRequest(repo, branchName))
	}
}
