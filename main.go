package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/config"
	"github.com/TouchBistro/cannon/git"
	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

var (
	configPath    string
	commitMessage string
)

func printRepos(repos []string) {
	fmt.Println("Affected repos:")
	for _, repo := range repos {
		fmt.Printf("- %s\n", repo)
	}
}

func cloneMissingRepos(repos []string) error {
	for _, repo := range repos {
		parts := strings.Split(repo, "/")
		orgName := parts[0]
		repoName := parts[1]
		path := fmt.Sprintf("%s/%s", config.CannonDir(), repoName)

		if util.FileOrDirExists(path) {
			fmt.Printf("Repo %s already exists\n", repo)
			continue
		}

		err := git.Clone(orgName, repoName, config.CannonDir())
		if err != nil {
			return errors.Wrapf(err, "failed to clone repo %s", repo)
		}

		fmt.Printf("Cloned repo %s to %s\n", repo, config.CannonDir())
	}

	return nil
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
	default:
		return errors.New(fmt.Sprintf("invalid action type %s", a.Type))
	}
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
		fmt.Fprintf(os.Stderr, "Failed reading config file. Error: %+v\n", err)
		os.Exit(1)
	}

	conf := config.Config()
	printRepos(conf.Repos)

	// Have user confirm changes
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm running with these parameters (y/n): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read user input. Error: %+v\n", err)
		os.Exit(1)
	}

	choice := strings.TrimSpace(input)
	if choice != "y" {
		fmt.Println("Aborting")
		os.Exit(0)
	}

	err = cloneMissingRepos(conf.Repos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to clone missing repos. Error: %+v\n", err)
		os.Exit(1)
	}

	// branchName := "cannon/change-" + uuid.NewV4().String()[0:8]

	// TODO apply changes
	// TODO make PR
}
