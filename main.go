package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/config"
	"github.com/TouchBistro/cannon/git"
	"github.com/TouchBistro/goutils/color"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	configPath    string
	commitMessage string
	noPush        bool
	noPR          bool
	verbose       bool
)

// Make sure repo is on master with latest changes
func prepareRepo(repo config.Repo) (*git.Repository, error) {
	path := filepath.Join(config.CannonDir(), repo.Name)

	// Repo doesn't exist, clone and then we are good to go
	if !file.FileOrDirExists(path) {
		log.Debugf("Repo %s does not exist, cloning...\n", repo.Name)

		r, err := git.Clone(repo.Name, config.CannonDir())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to clone repo %s", repo.Name)
		}

		log.Debugf("Cloned repo %s to %s\n", repo.Name, config.CannonDir())
		return r, nil
	}

	log.Debugf("Repo %s exists, updating...\n", repo.Name)
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

	log.Debugf("Updated repo %s\n", repo.Name)
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
	}

	return results, nil
}

func loadConfig() {
	var logLevel log.Level
	if verbose {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
		fatal.ShowStackTraces = false
	}

	log.SetLevel(logLevel)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})

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

	lock := sync.Mutex{}
	successCh := make(chan string)
	failedCh := make(chan error)

	// Clone or update each repo
	repositoryMap := make(map[string]*git.Repository)

	log.Info("☐ Preparing repos...")
	for _, r := range conf.Repos {
		log.Infof("\t☐ Preparing repos %s", r.Name)
		go func(repo config.Repo) {
			r, err := prepareRepo(repo)
			if err != nil {
				failedCh <- err
				return
			}

			err = git.CreateBranch(r, newBranchName, repo.Name)
			if err != nil {
				failedCh <- err
				return
			}

			lock.Lock()
			repositoryMap[repo.Name] = r
			lock.Unlock()
			successCh <- repo.Name
		}(r)
	}

	successMsg := color.Green("\t☑ Finished preparing repo %s\n")
	spinner.SpinnerWait(successCh, failedCh, successMsg, "failed preparing repo", len(conf.Repos))
	log.Info("☑ Finished preparing repos")

	// Execute actions for each repo
	resultsMap := make(map[string][]string)

	log.Info("☐ Running actions for repos...")
	for _, r := range conf.Repos {
		log.Infof(color.Cyan("\t☐ Running actions for repo %s"), r.Name)
		go func(repo config.Repo) {
			results, err := performActions(conf.Actions, repo)
			if err != nil {
				failedCh <- err
				return
			}

			lock.Lock()
			resultsMap[repo.Name] = results
			lock.Unlock()
			successCh <- repo.Name
		}(r)
	}

	successMsg = color.Green("\t☑ Finished running actions for repo %s\n")
	spinner.SpinnerWait(successCh, failedCh, successMsg, "failed to run actions for repo", len(conf.Repos))
	log.Info("☑ Finished running actions for repos")

	// Commit changes to each repo
	log.Info("☐ Committing changes to repos...")
	for _, repo := range conf.Repos {
		log.Infof(color.Cyan("\t☐ Committing changes to repo %s"), repo.Name)
		path := filepath.Join(config.CannonDir(), repo.Name)

		go func(name string, r *git.Repository) {
			err := git.Add(name, path, ".")
			if err != nil {
				failedCh <- err
				return
			}

			err = git.Commit(r, commitMessage, name)
			if err != nil {
				failedCh <- err
				return
			}

			successCh <- name
		}(repo.Name, repositoryMap[repo.Name])
	}

	successMsg = color.Green("\t☑ Finished committing changes to repo %s\n")
	spinner.SpinnerWait(successCh, failedCh, successMsg, "failed to commit changes to repo", len(conf.Repos))
	log.Info("☑ Finished committing changes repos")

	if noPush {
		os.Exit(0)
	}

	// Push local changes to remote and create PRs
	prURLs := make([]string, len(conf.Repos))

	log.Info("☐ Pushing changes to GitHub...")
	for i, repo := range conf.Repos {
		log.Infof("\t☐ Pushing changes for repo %s", repo.Name)
		r := repositoryMap[repo.Name]
		actionResults := resultsMap[repo.Name]

		go func(i int, repo config.Repo, r *git.Repository) {
			// Push changes to remote
			err := git.Push(r, repo.Name)
			if err != nil {
				failedCh <- err
				return
			}

			// Create pull requests or genreate pull request urls
			var url string
			if noPR {
				log.Debugf("Creating new PR URL for repo %s\n", repo.Name)
				url = git.CreatePRURL(repo.Name, newBranchName)
			} else {
				log.Debugf("Creating PR for repo %s\n", repo.Name)
				description := git.CreatePRDescription(actionResults)
				url, err = git.CreatePR(repo.Name, repo.BaseBranch(), newBranchName, description)
				if err != nil {
					failedCh <- err
					return
				}
			}

			lock.Lock()
			prURLs[i] = url
			lock.Unlock()

			successCh <- repo.Name
		}(i, repo, r)
	}

	successMsg = color.Green("\t☑ Finished pushing changes for repo %s\n")
	spinner.SpinnerWait(successCh, failedCh, successMsg, "failed to push changes for repo", len(conf.Repos))
	log.Info("☑ Finished pushing changes to GitHub")

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
	flag.BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	flag.Parse()
}
