package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/config"
	"github.com/TouchBistro/cannon/git"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
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
	path := filepath.Join(config.CannonDir(), repo.Name)

	// Repo doesn't exist, clone and then we are good to go
	if !file.FileOrDirExists(path) {
		fmt.Printf("Repo %s does not exist, cloning...", repo.Name)

		r, err := git.Clone(repo.Name, config.CannonDir())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to clone repo %s", repo.Name)
		}

		fmt.Printf("Cloned repo %s to %s\n", repo.Name, config.CannonDir())
		return r, nil
	}

	fmt.Printf("Repo %s exists, updating...\n", repo.Name)
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

	fmt.Printf("Updated repo %s\n", repo.Name)
	return r, errors.Wrapf(err, "failed to delete previous branch in repo %s", repo.Name)
}

func performActions(
	actions []action.Action,
	repo config.Repo,
) ([]string, error) {
	path := filepath.Join(config.CannonDir(), repo.Name)
	results := make([]string, len(actions))

	for i, a := range actions {
		var result string
		if a.IsLineAction() || a.IsTextAction() {
			filePath := filepath.Join(path, a.Path)
			file, err := os.OpenFile(filePath, os.O_RDWR, os.ModePerm)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to open file %s", filePath)
			}
			defer file.Close()

			result, err = action.ExecuteTextAction(a, file, file, repo.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to execute text action %s in repo %s", a.Type, repo.Name)
			}
		} else {
			var err error
			result, err = action.ExecuteFileAction(a, path, repo.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to execute file action %s in repo %s", a.Type, repo)
			}
		}

		results[i] = result
		fmt.Printf("  - %s\n", result)
	}

	return results, nil
}

func loadConfig() {
	if !file.FileOrDirExists(configPath) {
		fatal.Exitf("No such file %s", configPath)
	}

	file, err := os.Open(configPath)
	if err != nil {
		fatal.ExitErrf(err, "Failed to open config file %s", configPath)
	}
	defer file.Close()

	err = config.Init(file)
	if err != nil {
		fatal.ExitErr(err, "Failed reading config file.")
	}
}

func promptForConfirmation() {
	conf := config.Config()

	fmt.Println("Affected repos:")
	for _, repo := range conf.Repos {
		fmt.Printf("- %s\n", repo.Name)
	}

	fmt.Println()

	fmt.Println("Actions to perform:")
	for _, action := range conf.Actions {
		fmt.Printf("%s\n\n", action)
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

}

func main() {
	parseFlags()
	loadConfig()

	promptForConfirmation()

	fmt.Println()

	conf := config.Config()
	newBranchName := "cannon/change-" + uuid.NewV4().String()[0:8]

	// Clone or update each repo
	repositoryMap := make(map[string]*git.Repository)

	for _, repo := range conf.Repos {
		r, err := prepareRepo(repo)
		if err != nil {
			fatal.ExitErrf(err, "Failed to prepare repo %s", repo)
		}

		err = git.CreateBranch(r, newBranchName, repo.Name)
		if err != nil {
			fatal.ExitErrf(err, "Failed to create new branch %s in repo %s", newBranchName, repo.Name)
		}

		repositoryMap[repo.Name] = r
	}

	// Execute actions for each repo
	resultsMap := make(map[string][]string)

	for _, repo := range conf.Repos {
		fmt.Printf("%sRunning actions for repo %s%s\n", cyanColor, repo.Name, resetColor)

		results, err := performActions(conf.Actions, repo)
		if err != nil {
			fatal.ExitErrf(err, "Failed to perform actions on repo %s", repo)
		}

		resultsMap[repo.Name] = results

		fmt.Printf("%sSuccessfully performed actions for repo %s%s\n\n", greenColor, repo.Name, resetColor)
	}

	// Commit changes to each repo
	for _, repo := range conf.Repos {
		r := repositoryMap[repo.Name]
		path := filepath.Join(config.CannonDir(), repo.Name)

		err := git.Add(repo.Name, path, ".")
		if err != nil {
			fatal.ExitErrf(err, "failed to stage change files in repo %s", repo.Name)
		}

		err = git.Commit(r, commitMessage, repo.Name)
		if err != nil {
			fatal.ExitErrf(err, "failed to commit changes in repo %s", repo.Name)
		}
	}

	if noPush {
		os.Exit(0)
	}

	// Push local changes to remote and create PRs
	prURLs := make([]string, len(conf.Repos))

	for i, repo := range conf.Repos {
		r := repositoryMap[repo.Name]
		actionResults := resultsMap[repo.Name]

		// Push changes to remote
		err := git.Push(r, repo.Name)
		if err != nil {
			fatal.ExitErrf(err, "failed to push changes to remote for repo %s", repo.Name)
		}

		// Create pull requests or genreate pull request urls
		var url string
		if noPR {
			url = git.CreatePRURL(repo.Name, newBranchName)
		} else {
			description := git.CreatePRDescription(actionResults)
			url, err = git.CreatePR(repo.Name, repo.BaseBranch(), newBranchName, description)
			if err != nil {
				fatal.ExitErrf(err, "failed to create PR for repo %s", repo.Name)
			}
		}
		prURLs[i] = url
	}

	fmt.Println("Pull Request URLs:")
	for i, repo := range conf.Repos {
		fmt.Printf("- %s: %s\n", repo.Name, prURLs[i])
	}
}

func parseFlags() {
	flag.StringVarP(&configPath, "path", "p", "cannon.yml", "The path to a cannon.yml config file")
	flag.StringVarP(&commitMessage, "commit-message", "m", "Apply commit-cannon changes", "The commit message to use")
	flag.BoolVar(&noPush, "no-push", false, "Prevents pushing to remote repo")
	flag.BoolVar(&noPR, "no-pr", false, "Prevents creating a Pull Request in the remote repo")

	flag.Parse()
}
