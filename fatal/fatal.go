package fatal

import (
	"fmt"
	"os"
)

var ShowStackTraces = true

func ExitErr(err error, message string) {
	fmt.Fprintf(os.Stderr, message+"\n")

	if ShowStackTraces {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	}

	os.Exit(1)
}

func ExitErrf(err error, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Println()

	if ShowStackTraces {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
	}

	os.Exit(1)
}

func Exit(message string) {
	fmt.Fprintf(os.Stderr, message+"\n")
	os.Exit(1)
}

func Exitf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)

	os.Exit(1)
}
