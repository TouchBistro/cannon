package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/git"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var (
	configPath    string
	commitMessage string
	noPush        bool
	noPR          bool
	verbose       bool
)

func main() {
	flag.StringVarP(&configPath, "path", "p", "cannon.yml", "The path to a cannon.yml config file")
	flag.StringVarP(&commitMessage, "commit-message", "m", "Apply commit-cannon changes", "The commit message to use")
	flag.BoolVar(&noPush, "no-push", false, "Prevents pushing to remote repo")
	flag.BoolVar(&noPR, "no-pr", false, "Prevents creating a Pull Request in the remote repo")
	flag.BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	flag.Parse()

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		// Need to force colours since the decision of whether or not to use colour
		// is made lazily the first time a log is written, and Out may be changed
		// to a spinner before then.
		ForceColors: isatty.IsTerminal(os.Stderr.Fd()),
	})
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
		fatal.ShowStackTraces(true)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		fatal.ExitErr(err, "failed to get user's cache directory")
	}
	cannonDir := filepath.Join(cacheDir, "cannon")
	if err := os.MkdirAll(cannonDir, 0o755); err != nil {
		fatal.ExitErrf(err, "failed to create cannon directory at %s", cannonDir)
	}

	conf := readConfig()
	actions := make([]action.Action, len(conf.Actions))
	for i, c := range conf.Actions {
		a, err := action.Parse(c)
		if err != nil {
			fatal.ExitErrf(err, "failed to parse action config")
		}
		actions[i] = a
	}
	promptForConfirmation(conf.Repos, actions)

	newBranch := "cannon/change-" + uuid.NewV4().String()[0:8]
	repos := prepareRepos(conf.Repos, newBranch, cannonDir, logger)
	msgs := performActions(repos, actions, logger)
	commitChanges(repos, logger)
	logger.Info("Changes applied")
	if noPush {
		os.Exit(0)
	}

	prURLs := pushChanges(repos, msgs, newBranch, logger)
	fmt.Println("Pull Request URLs:")
	for i, repo := range repos {
		fmt.Printf("- %s: %s\n", repo.Name(), prURLs[i])
	}
}

type config struct {
	Repos   []repoConfig    `yaml:"repos"`
	Actions []action.Config `yaml:"actions"`
}

type repoConfig struct {
	Name string `yaml:"name"`
	Base string `yaml:"base"`
}

func readConfig() config {
	f, err := os.Open(configPath)
	if errors.Is(err, os.ErrNotExist) {
		fatal.Exitf("No such file %s", configPath)
	}
	if err != nil {
		fatal.ExitErrf(err, "Failed to open config file %s", configPath)
	}
	defer f.Close()

	var conf config
	err = yaml.NewDecoder(f).Decode(&conf)
	if err != nil {
		fatal.ExitErr(err, "Failed reading config file.")
	}
	for i, rc := range conf.Repos {
		if rc.Base == "" {
			conf.Repos[i].Base = "master"
		}
	}
	return conf
}

func promptForConfirmation(repos []repoConfig, actions []action.Action) {
	fmt.Println("Affected repos:")
	for _, repo := range repos {
		fmt.Printf("- %s\n", repo.Name)
	}
	fmt.Println("\nActions to perform:")
	for _, a := range actions {
		fmt.Printf("- %s\n\n", a)
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
}

func prepareRepos(repoConfigs []repoConfig, newBranch, cannonDir string, logger *logrus.Logger) []*git.Repository {
	s := spinner.New(
		spinner.WithStartMessage("Preparing repos"),
		spinner.WithCount(len(repoConfigs)),
		spinner.WithPersistMessages(verbose),
	)
	out := logger.Out
	logger.SetOutput(s)
	defer logger.SetOutput(out)
	s.Start()

	type repoResult struct {
		repo *git.Repository
		i    int
		err  error
	}
	resultCh := make(chan repoResult)
	for i, r := range repoConfigs {
		logger.Debugf("Preparing repo %s", r.Name)
		go func(i int, r repoConfig) {
			repo, err := git.Prepare(r.Name, cannonDir, r.Base, logger)
			if err != nil {
				resultCh <- repoResult{err: err}
				return
			}
			if err := repo.CreateBranch(newBranch); err != nil {
				resultCh <- repoResult{err: err}
				return
			}
			resultCh <- repoResult{repo: repo, i: i}
		}(i, r)
	}

	repos := make([]*git.Repository, len(repoConfigs))
	for i := 0; i < len(repoConfigs); i++ {
		select {
		case res := <-resultCh:
			if res.err != nil {
				s.Stop()
				fatal.ExitErr(res.err, "failed preparing repo")
			}
			repos[res.i] = res.repo
			s.Inc()
		case <-time.After(5 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while preparing repos")
		}
	}
	s.Stop()
	return repos
}

func performActions(repos []*git.Repository, actions []action.Action, logger *logrus.Logger) [][]string {
	s := spinner.New(
		spinner.WithStartMessage("Running actions on repos"),
		spinner.WithCount(len(repos)),
		spinner.WithPersistMessages(verbose),
	)
	out := logger.Out
	logger.SetOutput(s)
	defer logger.SetOutput(out)
	s.Start()

	type result struct {
		i    int
		msgs []string
		err  error
	}
	resultCh := make(chan result)
	for i, repo := range repos {
		logger.Debugf("Running actions on repo %s", repo.Name())
		go func(i int, repo *git.Repository) {
			res := result{i: i}
			parts := strings.Split(repo.Name(), "/")
			// Variables that will be shared across all actions
			vars := map[string]string{
				"REPO_OWNER": parts[0],
				"REPO_NAME":  parts[1],
			}
			args := action.Arguments{Variables: vars}
			for _, a := range actions {
				msg, err := a.Run(repo, args)
				if err != nil {
					res.err = err
					resultCh <- res
					return
				}
				res.msgs = append(res.msgs, msg)
			}
			resultCh <- res
		}(i, repo)
	}

	msgs := make([][]string, len(repos))
	for i := 0; i < len(repos); i++ {
		select {
		case res := <-resultCh:
			if res.err != nil {
				s.Stop()
				fatal.ExitErr(res.err, "failed running actions on repo")
			}
			msgs[res.i] = res.msgs
			s.Inc()
		case <-time.After(5 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while running actions")
		}
	}
	s.Stop()
	return msgs
}

func commitChanges(repos []*git.Repository, logger *logrus.Logger) {
	s := spinner.New(
		spinner.WithStartMessage("Committing changes to repos"),
		spinner.WithCount(len(repos)),
		spinner.WithPersistMessages(verbose),
	)
	out := logger.Out
	logger.SetOutput(s)
	defer logger.SetOutput(out)
	s.Start()

	resultCh := make(chan error)
	for _, repo := range repos {
		logger.Debugf("Committing changes to repo %s", repo.Name())
		go func(repo *git.Repository) {
			resultCh <- repo.CommitChanges(commitMessage)
		}(repo)
	}
	for i := 0; i < len(repos); i++ {
		select {
		case err := <-resultCh:
			if err != nil {
				s.Stop()
				fatal.ExitErr(err, "failed to commit changes to repo")
			}
			s.Inc()
		case <-time.After(5 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while committing changes")
		}
	}
	s.Stop()
}

func pushChanges(repos []*git.Repository, msgs [][]string, newBranch string, logger *logrus.Logger) []string {
	s := spinner.New(
		spinner.WithStartMessage("Pushing changes to GitHub"),
		spinner.WithCount(len(repos)),
		spinner.WithPersistMessages(verbose),
	)
	out := logger.Out
	logger.SetOutput(s)
	defer logger.SetOutput(out)
	s.Start()

	type result struct {
		i   int
		url string
		err error
	}
	resultCh := make(chan result)
	for i, repo := range repos {
		logger.Debugf("Pushing changes for repo %s", repo.Name())
		go func(i int, repo *git.Repository) {
			res := result{i: i}
			if err := repo.Push(); err != nil {
				res.err = err
				resultCh <- res
				return
			}
			if noPR {
				res.url = git.CreatePRURL(repo.Name(), newBranch)
				resultCh <- res
				return
			}

			logger.Debugf("Creating PR for repo %s", repo.Name())
			var desc strings.Builder
			desc.WriteString("Changes applied by commit-cannon:\n")
			for _, m := range msgs[i] {
				desc.WriteString("  * ")
				desc.WriteString(m)
				desc.WriteByte('\n')
			}
			res.url, res.err = repo.CreatePR(newBranch, desc.String())
			resultCh <- res
		}(i, repo)
	}

	urls := make([]string, len(repos))
	for i := 0; i < len(repos); i++ {
		select {
		case res := <-resultCh:
			if res.err != nil {
				s.Stop()
				fatal.ExitErr(res.err, "failed to push changes for repo")
			}
			urls[res.i] = res.url
			s.Inc()
		case <-time.After(5 * time.Minute):
			s.Stop()
			fatal.Exit("Timed out while pushing changes")
		}
	}
	s.Stop()
	return urls
}
