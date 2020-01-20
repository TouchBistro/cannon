package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/config"
	"github.com/TouchBistro/cannon/fatal"
	"github.com/TouchBistro/cannon/git"
	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	flag "github.com/spf13/pflag"
)

const (
	greenColor = "\x1b[32m"
	cyanColor  = "\x1b[36m"
	resetColor = "\x1b[0m"
)

var (
	configPath    string
	commitMessage string
	noPush        bool
	noPR          bool
)

// Make sure repo is on master with latest changes
func prepareRepo(repo config.Repo) (*git.Repository, error) {
	parts := strings.Split(repo.Name, "/")
	orgName := parts[0]
	repoName := parts[1]
	path := fmt.Sprintf("%s/%s", config.CannonDir(), repoName)

	// Repo doesn't exist, clone and then we are good to go
	if !util.FileOrDirExists(path) {
		fmt.Printf("Repo %s does not exist, cloning...", repo.Name)

		r, err := git.Clone(orgName, repoName, config.CannonDir())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to clone repo %s", repo.Name)
		}

		fmt.Printf("Cloned repo %s to %s\n", repo.Name, config.CannonDir())
		return r, nil
	}

	fmt.Printf("Repo %s exits, updating...", repo.Name)
	r, err := git.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open repo %s", repo.Name)
	}

	branchRef, err := r.Head()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get HEAD for repo %s", repo.Name)
	}

	// Discard any changes and switch to base branch
	baseBranch := repo.BaseBranch()
	err = git.CleanAndCheckout(r, baseBranch, repo.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to clean up repo %s", repo.Name)
	}

	// Pull latest changes
	err = git.Pull(r, repo.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update %s branch for repo %s", baseBranch, repo.Name)
	}

	// Check if already on base branch
	if branchRef.Name().Short() == baseBranch {
		return r, nil
	}

	// Delete old branch
	err = git.DeleteBranch(r, branchRef.Name().Short(), repo.Name)

	fmt.Printf("Updated repo %s", repo.Name)
	return r, errors.Wrapf(err, "failed to delete previous branch in repo %s", repo.Name)
}

func performActions(
	r *git.Repository,
	actions []action.Action,
	branchName string,
	repo config.Repo,
) (string, error) {
	err := git.CreateBranch(r, branchName, repo.Name)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create new branch %s in repo %s", branchName, repo.Name)
	}

	// Execute actions
	repoName := strings.Split(repo.Name, "/")[1]
	path := fmt.Sprintf("%s/%s", config.CannonDir(), repoName)
	results := make([]string, len(actions))
	for i, a := range actions {
		result, err := action.ExecuteAction(a, path, repoName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to execute action %s in repo %s", a.Type, repo.Name)
		}

		results[i] = result
		fmt.Printf("  - %s\n", result)
	}

	// Commit changes and push
	err = git.Add(repoName, path, ".")
	if err != nil {
		return "", errors.Wrapf(err, "failed to stage change files in repo %s", repo.Name)
	}

	err = git.Commit(r, commitMessage, repo.Name)
	if err != nil {
		return "", errors.Wrapf(err, "failed to commit changes in repo %s", repo.Name)
	}

	if noPush {
		return "", nil
	}

	err = git.Push(r, repo.Name)
	if err != nil {
		return "", errors.Wrapf(err, "failed to push changes to remote for repo %s", repo.Name)
	}

	if noPR {
		return git.CreatePRURL(repo.Name, branchName), nil
	}

	return git.CreatePR(repo.Name, repo.BaseBranch(), branchName, git.CreatePRDescription(results))
}

func parseFlags() {
	flag.StringVarP(&configPath, "path", "p", "cannon.yml", "The path to a cannon.yml config file")
	flag.StringVarP(&commitMessage, "commit-message", "m", "", "The commit message to use")
	flag.BoolVar(&noPush, "no-push", false, "Prevents pushing to remote repo")
	flag.BoolVar(&noPR, "no-pr", false, "Prevents creating a Pull Request in the remote repo")

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
		fmt.Printf("- %s\n", repo.Name)
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
	prURLs := make([]string, len(conf.Repos))

	// Make sure repos are up to date
	for i, repo := range conf.Repos {
		r, err := prepareRepo(repo)

		fmt.Printf("%sRunning actions for repo %s%s\n", cyanColor, repo.Name, resetColor)
		if err != nil {
			fatal.ExitErrf(err, "Failed to prepare repo %s.", repo)
		}

		url, err := performActions(r, conf.Actions, branchName, repo)
		if err != nil {
			fatal.ExitErrf(err, "Failed to perform actions on repo %s.", repo)
		}
		prURLs[i] = url

		fmt.Printf("%sSuccessfully performed actions for repo %s%s\n\n", greenColor, repo.Name, resetColor)
	}

	// No point in printing anything PR related if we didn't push
	if noPush {
		return
	}

	fmt.Println("Pull Request URLs:")
	for i, repo := range conf.Repos {
		fmt.Printf("- %s: %s\n", repo.Name, prURLs[i])
	}
}
