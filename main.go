package main

import (
	_ "embed"
	"os"

	"github.com/voidwyrm-2/rush/cmd"
)

//go:embed version.txt
var version string

func main() {
	if err := cmd.Execute(version); err != nil {
		os.Exit(1)
	}
}
