package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/git"
	"github.com/TouchBistro/goutils/fatal"
	"github.com/TouchBistro/goutils/log"
	"github.com/TouchBistro/goutils/progress"
	"github.com/TouchBistro/goutils/spinner"
	"github.com/mattn/go-isatty"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type options struct {
	configPath string
	commitMsg  string
	noPush     bool
	noPR       bool
	verbose    bool
	clean      bool
}

func main() {
	if err := execute(); err != nil {
		fatal.PrintAndExit(err)
	}
}

func execute() error {
	var opts options
	flag.StringVarP(&opts.configPath, "path", "p", "cannon.yml", "The path to a cannon.yml config file")
	flag.StringVarP(&opts.commitMsg, "commit-message", "m", "Apply commit-cannon changes", "The commit message to use")
	flag.BoolVar(&opts.noPush, "no-push", false, "Prevents pushing to remote repo")
	flag.BoolVar(&opts.noPR, "no-pr", false, "Prevents creating a Pull Request in the remote repo")
	flag.BoolVarP(&opts.verbose, "verbose", "v", false, "Enable verbose logging")
	flag.BoolVar(&opts.clean, "clean", false, "Clean cannon cache directory")
	flag.Parse()

	level := log.LevelInfo
	if opts.verbose {
		level = log.LevelDebug
	}
	logger := log.New(
		log.WithFormatter(&log.TextFormatter{
			Pretty: isatty.IsTerminal(os.Stderr.Fd()),
		}),
		log.WithLevel(level),
	)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get user's cache directory: %w", err)
	}
	cannonDir := filepath.Join(cacheDir, "cannon")
	if opts.clean {
		if err := os.RemoveAll(cannonDir); err != nil {
			return fmt.Errorf("failed to clean cannon directory at %s: %w", cannonDir, err)
		}
	}
	if err := os.MkdirAll(cannonDir, 0o755); err != nil {
		return fmt.Errorf("failed to create cannon directory at %s: %w", cannonDir, err)
	}

	conf, err := readConfig(opts.configPath)
	if err != nil {
		return err
	}
	actions := make([]action.Action, len(conf.Actions))
	for i, c := range conf.Actions {
		a, err := action.Parse(c)
		if err != nil {
			return fmt.Errorf("failed to parse action config: %w", err)
		}
		actions[i] = a
	}

	// Show the actions that will be performed to the user and prompt for confirmation before proceeding.
	fmt.Println("Affected repos:")
	for _, repo := range conf.Repos {
		fmt.Printf("- %s\n", repo.Name)
	}
	fmt.Println("\nActions to perform:")
	for _, a := range actions {
		fmt.Printf("- %s\n\n", a)
	}
	// Read the user's response
	fmt.Print("\nConfirm running with these parameters (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}
	// Support Y/y, everything else is no.
	if strings.ToLower(strings.TrimSpace(input)) != "y" {
		fmt.Println("Aborting")
		return nil
	}
	fmt.Println()

	tracker := &spinner.Tracker{
		OutputLogger:    logger,
		PersistMessages: opts.verbose,
	}
	ctx := progress.ContextWithTracker(context.Background(), tracker)
	// Create a random number suffix so each branch is unique.
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("failed to generate random branch number: %w", err)
	}
	newBranch := "cannon/change-" + hex.EncodeToString(b)

	repos, err := progress.RunParallelT(ctx, progress.RunParallelOptions{
		Message: "Preparing repos",
		Count:   len(conf.Repos),
	}, func(ctx context.Context, i int) (*git.Repository, error) {
		r := conf.Repos[i]
		tracker := progress.TrackerFromContext(ctx)
		tracker.Debugf("Preparing repo %s", r.Name)

		repo, err := git.Prepare(r.Name, cannonDir, r.Base, tracker)
		if err != nil {
			return nil, err
		}
		if err := repo.CreateBranch(newBranch); err != nil {
			return nil, err
		}
		return repo, nil
	})
	if err != nil {
		return fmt.Errorf("failed to prepare repos: %w", err)
	}

	type actionResult struct {
		repoIndex int
		msgs      []string
	}

	actionResults, err := progress.RunParallelT(ctx, progress.RunParallelOptions{
		Message:       "Running actions on repos",
		Count:         len(repos),
		CancelOnError: true,
	}, func(ctx context.Context, i int) (actionResult, error) {
		repo := repos[i]

		tracker := progress.TrackerFromContext(ctx)
		tracker.Debugf("Running actions on repo %s", repo.Name())

		parts := strings.Split(repo.Name(), "/")
		// Variables that will be shared across all actions
		vars := map[string]string{
			"REPO_OWNER": parts[0],
			"REPO_NAME":  parts[1],
		}
		args := action.Arguments{Variables: vars}
		res := actionResult{repoIndex: i, msgs: make([]string, 0, len(actions))}
		for _, a := range actions {
			msg, err := a.Run(ctx, repo, args)
			if err != nil {
				return res, err
			}
			res.msgs = append(res.msgs, msg)
		}
		return res, nil
	})
	if err != nil {
		return fmt.Errorf("failed to perform actions on repos: %w", err)
	}
	repoMsgs := make(map[int][]string)
	for _, res := range actionResults {
		repoMsgs[res.repoIndex] = res.msgs
	}

	err = progress.RunParallel(ctx, progress.RunParallelOptions{
		Message:       "Committing changes to repos",
		Count:         len(repos),
		CancelOnError: true,
	}, func(ctx context.Context, i int) error {
		repo := repos[i]
		tracker := progress.TrackerFromContext(ctx)
		tracker.Debugf("Committing changes to repo %s", repo.Name())
		return repo.CommitChanges(opts.commitMsg)
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes to repos: %w", err)
	}

	logger.Info("Changes applied")
	if opts.noPush {
		return nil
	}

	prURLs, err := progress.RunParallelT(ctx, progress.RunParallelOptions{
		Message: "Pushing changes to GitHub",
		Count:   len(repos),
	}, func(ctx context.Context, i int) (string, error) {
		repo := repos[i]
		tracker := progress.TrackerFromContext(ctx)
		tracker.Debugf("Pushing changes for repo %s", repo.Name())

		if err := repo.Push(); err != nil {
			return "", err
		}
		if opts.noPR {
			return git.CreatePRURL(repo.Name(), newBranch), nil
		}

		tracker.Debugf("Creating PR for repo %s", repo.Name())
		var desc strings.Builder
		desc.WriteString("Changes applied by commit-cannon:\n")
		msgs, ok := repoMsgs[i]
		if !ok {
			return "", fmt.Errorf("impossible: no messages found for repo %s", repo.Name())
		}
		for _, m := range msgs {
			desc.WriteString("  * ")
			desc.WriteString(m)
			desc.WriteByte('\n')
		}
		return repo.CreatePR(newBranch, desc.String())
	})
	if err != nil {
		return fmt.Errorf("failed to push changes to repos: %w", err)
	}
	fmt.Println("Pull Request URLs:")
	for i, repo := range repos {
		fmt.Printf("- %s: %s\n", repo.Name(), prURLs[i])
	}
	return nil
}

type config struct {
	Repos   []repoConfig    `yaml:"repos"`
	Actions []action.Config `yaml:"actions"`
}

type repoConfig struct {
	Name string `yaml:"name"`
	Base string `yaml:"base"`
}

func readConfig(configPath string) (config, error) {
	f, err := os.Open(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return config{}, fmt.Errorf("no such file %s", configPath)
	}
	if err != nil {
		return config{}, fmt.Errorf("failed to open config file %s: %w", configPath, err)
	}
	defer f.Close()

	var conf config
	err = yaml.NewDecoder(f).Decode(&conf)
	if err != nil {
		return conf, fmt.Errorf("failed to read config file: %w", err)
	}
	for i, rc := range conf.Repos {
		if rc.Base == "" {
			conf.Repos[i].Base = "master"
		}
	}
	return conf, nil
}
