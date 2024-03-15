package main

import (
	"os"

	lrs "github.com/handlename/let-rds-sleep"
)

func main() {
	os.Exit(lrs.RunCLI())
}
