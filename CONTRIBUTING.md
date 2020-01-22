# Contributing

The following document outlines how to contribute to the `cannon` project.

### **Table of Contents**
- [Requirements](#requirements)
- [Setup](#setup)
- [Building](#building)
    + [Running locally](#running-locally)
    + [Running globally](#running-globally)
    + [Remove global build](#remove-global-build)
    + [Running tests](#running-tests)

## Requirements

To build and run `cannon` locally you will need to install go.
This can likely be done through your package manager or by going to https://golang.org/doc/install.

## Setup
First clone the repo to your desired location:
```sh
git clone git@github.com:TouchBistro/cannon.git
```

Then in the root of the repo run the following to install all dependencies and tools required:
```sh
make setup
```

## Building
### Running locally
To build the app run:
```sh
go build
```

This will create a binary named `cannon` in the current directory. You can run it be doing `./cannon`.

### Running globally
If you want to be able to run if from anywhere you can run:
```sh
go install
```

This will installing it in the `bin` directory in your go path.

**NOTE:** You will need to add the go bin to your `PATH` variable.
Add the following to your `.zshrc` or `.bash_profile`:
```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Then run `source ~/.zshrc` or `source ~/.bash_profile`.

### Remove global build
To remove the globally installed build run the following from the root directory of the repo:
```sh
make go-uninstall
```

### Running tests
To run the tests run:
```sh
make test
```

This will output code coverage information to the `coverage` directory.
You can open the `coverage/coverage.html` file in your browser to visually see the coverage in each file.
