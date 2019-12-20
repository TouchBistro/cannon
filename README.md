# cannon

cannon is a small CLI tool that lets you make changes to multiple git repos.

## Setup Instructions

1. Make sure you have go installed and set up.
2. Clone the repo:
    ```sh
    git clone git@github.com:touchbistro/cannon.git
    ```
3. Compile and install cannon globally:
    ```sh
    go install cannon
    ```
4. Create a GitHub Access Token
    - Create the token with the `repo` box checked in the list of permissions. Follow the instructions [here](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) to learn more.
        - Make sure you copy the token when you create it!
    - After the token has been created, enable SSO for it.
    - Add the following to your `.bash_profile` or `.zshrc`:
    ```sh
    export HOMEBREW_GITHUB_API_TOKEN=<YOUR_TOKEN>
    ```
    - Run `source ~/.zshrc` or `source ~/.bash_profile`.

## Usage
To use cannon provide a `cannon.yml` file which contains a list of repos and a list of actions to apply.

```sh
Usage of ./cannon:
  -m, --commit-message string   The commit message to use
      --no-pr                   Prevents creating a Pull Request in the remote repo
      --no-push                 Prevents pushing to remote repo
  -p, --path string             The path to a cannon.yml config file (default "cannon.yml")
```

cannon supports the following actions:
1. replaceLine - Replace an entire line including any leading whitespace at specified path (trims trailing whitespace).
    ```yml
    type: replaceLine
    source: <The line to add>
    target: <The line that will be replaced>
    path: <The path to the file>
    ```
2. deleteLine - Delete an entire line including any leading whitespace at the specified path.
    ```yml
    type: deleteLine
    target: <The line to delete>
    path: <The path to the file>
    ```
3. replaceText - Replace some text at specified path.
    ```yml
    type: replaceText
    source: <The text to add>
    target: <The text that will be replaced>
    path: <The path to the file>
    ```
4. appendText - Append text to matching text.
    ```yml
    type: appendText
    source: <The text to append>
    target: <The text that will be appended to>
    path: <The path to the file> 
    ```
5. createFile - Create file at specified path if it doesn't already exist.
    ```yml
    type: createFile
    source: <The file to use>
    path: <The path to create the file at>
    ```
6. deleteFile - Delete file at specified path if it already exists.
    ```yml
    type: deleteFile
    path: <The path to the file to delete>
    ```
7. replaceFile - Replace file at specified path if it already exists.
    ```yml
    type: replaceFile
    source: <The file to use>
    path: <The path to the file to replace>
    ```
8. createOrReplaceFile - Create or replace file at specified path.
    ```yml
    type: createOrReplaceFile
    source: <The file to use>
    path: <The path to create or replace the file at>
    ```
9. runCommand - Runs a given shell command in the repo.
    ```yml
    type: runCommand
    run: <The shell command to run>
    ```
    You can also run shell (`sh`) commands by prefixing the command with `SHELL >>`.  
    Ex:
    ```yml
    type: runCommand
    run: SHELL >> if [[ ! -d data ]]; then mkdir data; touch data/.gitkeep; fi
    ```

## Configuration

`cannon.yml` example:
The `cannon.yml` file is structured as follows:
```yml
repos:
  - name: TouchBistro/touchbistro-node-boilerplate
  - name: TouchBistro/ordering-deliveroo-service
actions:
  - type: replaceLine
    source: DB_USER=core
    target: DB_USER=SA
    path: .env.example
  - type: replaceText
    source: LOGGER.debug
    target: console.log
    path: src/index.ts
  - type: createFile
    source: files/text.txt
    path: text.txt
  - type: deleteFile
    path: .env.example
  - type: replaceFile
    source: files/.env.test
    path: .env.compose
  - type: createOrReplaceFile
    source: files/.env.test
    path: .env.test
  - type: runCommand
    run: yarn
```

### Change base branch for PRs

By default `cannon` will target `master` as the base branch when creating PRs.
If you need to use a different base branch you can set it using the `base` field of the repo.

Example:
```yml
repos:
  - name: TouchBistro/repo-name
    base: develop
```

This would create PRs with `develop` as the base branch.
