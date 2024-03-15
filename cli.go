package lrs

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	version string
)

const (
	ExitStatusOK    = 0
	ExitStatusError = 1
)

func RunCLI() int {
	var (
		flagMode    string
		flagTarget  string
		flagExclude string
		flagDryRun  bool
		flagVersion bool
	)

	flag.StringVar(&flagMode, "mode", "", "STOP or START, only for oneshot mode")
	flag.StringVar(&flagTarget, "target", "", "TagName=Value,... If no tags given, treat all of resources as target")
	flag.StringVar(&flagExclude, "exclude", "", "TagName=Value,... If Tag exists exclude the resource")
	flag.BoolVar(&flagDryRun, "dryrun", false, "show process target only")
	flag.BoolVar(&flagVersion, "version", false, "display version")
	flag.Parse()

	if flagVersion {
		fmt.Printf("let-rds-sleep %s", version)
		return ExitStatusError
	}

	app, err := New(
		OptTarget(flagTarget),
		OptExclude(flagExclude),
		OptDryRun(flagDryRun),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init app: %s", err)
		return ExitStatusError
	}

	if strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") || os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		if flagMode != "" {
			fmt.Fprintf(os.Stderr, "-mode option is not available in Lambda mode")
		}

		fmt.Println("started as Lambda function handler")
		lambda.Start(app.HandleRequest)
	} else {
		fmt.Println("started as oneshot mode")

		var mode string
		switch strings.ToUpper(flagMode) {
		case "STOP":
			mode = ModeStop
		case "START":
			mode = ModeStart
		default:
			fmt.Fprintf(os.Stderr, "invalid mode: %s", flagMode)
			return ExitStatusError
		}

		if err := app.Run(context.Background(), mode); err != nil {
			fmt.Fprintf(os.Stderr, "failed to Run: %s", err)
			return ExitStatusError
		}
	}

	fmt.Println("bye")

	return ExitStatusOK
}

func parseCLIArgs() {

}
