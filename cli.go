package lrs

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

const (
	ExitStatusOK    = 0
	ExitStatusError = 1
)

type CLIFlags struct {
	Mode    string
	Target  string
	Exclude string
	DryRun  bool
	Version bool
}

func RunCLI() int {
	flags, err := parseCLIFlags(os.Args[1:])
	if err != nil {
		log.Printf("failed to parse flags: %s", err)
		return ExitStatusError
	}

	if err := validateCLIFlags(flags); err != nil {
		log.Printf("failed to validate flags: %s", err)
		return ExitStatusError
	}

	if flags.Version {
		fmt.Printf("let-rds-sleep %s", version)
		return ExitStatusOK
	}

	if err := runApp(flags); err != nil {
		log.Printf("[INFO] failed to run: %s", err)
		return ExitStatusError
	}

	log.Println("[INFO] bye")

	return ExitStatusOK
}

func isLambda() bool {
	return strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") ||
		os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
}

func parseCLIFlags(args []string) (*CLIFlags, error) {
	flags := &CLIFlags{}

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fs.StringVar(&flags.Mode, "mode", "", "STOP or START, only for oneshot mode")
	fs.StringVar(&flags.Target, "target", "", "TagName=Value,... If no tags given, treat all of resources as target")
	fs.StringVar(&flags.Exclude, "exclude", "", "TagName=Value,... If Tag exists exclude the resource")
	fs.BoolVar(&flags.DryRun, "dryrun", false, "show process target only")
	fs.BoolVar(&flags.Version, "version", false, "display version")

	// overwrite by environment variables
	fs.VisitAll(func(f *flag.Flag) {
		if env := getEnv(strings.ToUpper(f.Name)); env != "" {
			f.Value.Set(env)
		}
	})

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return flags, nil
}

func validateCLIFlags(flags *CLIFlags) error {
	if isLambda() && flags.Mode != "" {
		return fmt.Errorf("-mode option is only available in oneshot mode")
	}

	return nil
}

func runApp(flags *CLIFlags) error {
	app, err := New(
		OptTarget(flags.Target),
		OptExclude(flags.Exclude),
		OptDryRun(flags.DryRun),
	)
	if err != nil {
		return fmt.Errorf("failed to init app: %s", err)
	}

	if isLambda() {
		log.Println("[INFO] started as Lambda function handler")
		lambda.Start(app.HandleRequest)
	} else {
		log.Println("[INFO] started as oneshot mode")

		var mode string
		switch strings.ToUpper(flags.Mode) {
		case "STOP":
			mode = ModeStop
		case "START":
			mode = ModeStart
		default:
			return fmt.Errorf("invalid mode: %s", flags.Mode)
		}

		if err := app.Run(context.Background(), mode); err != nil {
			return fmt.Errorf("failed to Run: %s", err)
		}
	}

	return nil
}
