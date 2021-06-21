# Commit Cannon

`cannon` is a small CLI tool that lets you make changes to multiple git repos.

![](docs/resources/cannon.gif)

## Why?

Suppose you need to make the same change across multiple git repos.
You could do it all by hand but this will get quite tedious especially with the more repos you have. Automation to the rescue!

`cannon` makes it easy to perform a batch of changes on multiple git repos at once. It even creates GitHub PRs by default.
All the heavy lifting is taken care of giving you time to do more important things.

If you want to know more about why we did this and see a use case for it checkout out our [blog post](https://medium.com/touchbistro-development/commit-cannon-open-source-project-899ee75794c0).

## Setup Instructions

1. Make sure you have [Go](https://golang.org/doc/install) installed and set up.
2. Install `cannon`:

   ```sh
   go install github.com/TouchBistro/cannon@latest
   ```

   **NOTE:** Make sure you have the following in your `~/.bash_profile` or `~/.zshrc` to ensure programs installed with `go install` are available globally:

   ```sh
   export PATH="$(go env GOPATH)/bin:$PATH"
   ```

3. Create a GitHub Access Token:

   A Github Access Token is required to be able create GitHub PRs for changes to repos.

   - Create the token with the `repo` box checked in the list of permissions. Follow the instructions [here](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) to learn more.
   - Set the `GITHUB_TOKEN` environment variable to the value of the token.
     For example if you use bash or zsh, add the following to your `.bash_profile` or `.zshrc`:

     ```sh
     export GITHUB_TOKEN=<YOUR_TOKEN>
     ```

## Usage

To use `cannon` provide a `cannon.yml` file which contains a list of repos and a list of actions to apply.

```sh
Usage of ./cannon:
      --clean                   Clean cannon cache directory
  -m, --commit-message string   The commit message to use (default "Apply commit-cannon changes")
      --no-pr                   Prevents creating a Pull Request in the remote repo
      --no-push                 Prevents pushing to remote repo
  -p, --path string             The path to a cannon.yml config file (default "cannon.yml")
  -v, --verbose                 Enable verbose logging
```

### Actions

`cannon` supports 3 categories of actions which are described below.

#### Text Actions

A text action is an action applied to the text in a file within a repo.
The search text for the action is a regex which allows for complex searches within a file.
The `path` field in the config must be relative to the root of the repo.

The following text actions are supported:

1. `replaceLine` - Replace an entire line including any leading whitespace (trims trailing whitespace).
   ```yml
   type: replaceLine
   searchText: <The line that will be replaced>
   applyText: <The line to add>
   path: <The path to the file>
   ```
2. `deleteLine` - Delete an entire line including any leading whitespace.
   ```yml
   type: deleteLine
   searchText: <The line to delete>
   path: <The path to the file>
   ```
3. `replaceText` - Replace some text. The text can span multiple lines.
   ```yml
   type: replaceText
   searchText: <The text that will be replaced>
   applyText: <The text to add>
   path: <The path to the file>
   ```
4. `appendText` - Append text to matching text.
   ```yml
   type: appendText
   searchText: <The text that will be appended to>
   applyText: <The text to append>
   path: <The path to the file>
   ```
5. `deleteText` - Delete matching text. The text can span multiple lines.
   ```yml
   type: deleteText
   searchText: <The text to delete>
   path: <The path to the file>
   ```

#### File Actions

A file action is an action applied to an entire file within a repo.
The `dstPath` field in the config must be relative to the root of the repo.

The following file actions are supported:

1. `createFile` - Create a file if it doesn't already exist.
   ```yml
   type: createFile
   srcPath: <The file to use>
   dstPath: <The path to create the file at>
   ```
2. `replaceFile` - Replace a file if it already exists.
   ```yml
   type: replaceFile
   srcPath: <The file to use>
   dstPath: <The path to the file to replace>
   ```
3. `createOrReplaceFile` - Create a file if it doesn't exist or replace it if it does exist.
   ```yml
   type: createOrReplaceFile
   srcPath: <The file to use>
   dstPath: <The path to create or replace the file at>
   ```
4. `deleteFile` - Delete a file if it already exists.
   ```yml
   type: deleteFile
   dstPath: <The path to the file to delete>
   ```

#### Command Action

A command action allows for running a command in a repo.

The following command actions are supported:

1. `runCommand` - Runs a command in the repo.
   ```yml
   type: runCommand
   run: <The command to run>
   ```
2. `shellCommand` - Runs a command in a shell (`sh`) in a repo.
   ```yml
   type: shellCommand
   run: <The shell command to run>
   ```

## Configuration

`cannon.yml` example:
The `cannon.yml` file is structured as follows:

```yml
repos:
  - name: TouchBistro/touchbistro-node-boilerplate
  - name: TouchBistro/touchbistro-node-shared
actions:
  - type: replaceLine
    searchText: DB_USER=SA
    applyText: DB_USER=core
    path: .env.example
  - type: replaceText
    searchText: console.log
    applyText: LOGGER.debug
    path: src/index.ts
  - type: createFile
    srcPath: files/text.txt
    dstPath: text.txt
  - type: deleteFile
    dstPath: .env.example
  - type: replaceFile
    srcPath: files/.env.test
    dstPath: .env.compose
  - type: createOrReplaceFile
    srcPath: files/.env.test
    dstPath: .env.test
  - type: runCommand
    run: yarn install
  - type: shellCommand
    run: if [ ! -d data ]; then mkdir data; touch data/.gitkeep; fi
```

### Change base branch for PRs

By default `cannon` will target `master` as the base branch when creating PRs.
If you need to use a different base branch you can set it using the `base` field of the repo.

Example:

```yml
repos:
  - name: org/repo-name
    base: develop
```

This would create PRs with `develop` as the base branch.

## Contributing

See [contributing](CONTRIBUTING.md) for instructions on how to contribute to `cannon`. PRs welcome!

## License

MIT © TouchBistro, see [LICENSE](LICENSE) for details.
