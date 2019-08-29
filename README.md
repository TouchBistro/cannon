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

## Usage
To use cannon provide a `cannon.yml` file which contains a list of repos and a list of actions to apply.

```sh
Usage of ./cannon:
  -m, --commit-message string   The commit message to use
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
2. replaceText - Replace some text at specified path.
    ```yml
    type: replaceText
    source: <The text to add>
    target: <The text that will be replaced>
    path: <The path to the file>
    ```
3. createFile - Create file at specified path if it doesn't already exist.
    ```yml
    type: createFile
    source: <The file to use>
    path: <The path to create the file at>
    ```
4. deleteFile - Delete file at specified path if it already exists.
    ```yml
    type: deleteFile
    path: <The path to the file to delete>
    ```
5. replaceFile - Replace file at specified path if it already exists.
    ```yml
    type: replaceFile
    source: <The file to use>
    path: <The path to the file to replace>
    ```
6. createOrReplaceFile - Create or replace file at specified path.
    ```yml
    type: createOrReplaceFile
    source: <The file to use>
    path: <The path to create or replace the file at>
    ```
7. runCommand - Runs a given shell command in the repo.
    ```yml
    type: runCommand
    run: <The shell command to run>
    ```

`cannon.yml` example:
The `cannon.yml` file is structured as follows:
```yml
repos:
  - TouchBistro/touchbistro-node-boilerplate
  - TouchBistro/ordering-deliveroo-service
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
