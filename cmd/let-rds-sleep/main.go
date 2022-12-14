package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	lrs "github.com/handlename/let-rds-sleep"
)

var (
	version string
)

func main() {
	var (
		flagMode    string
		flagTarget  string
		flagExclude string
		flagDryRun  bool
		flagVersion bool
	)

	flag.StringVar(&flagMode, "mode", "", "STOP or START")
	flag.StringVar(&flagTarget, "target", "", "TagName=Value,... If no tags given, treat all of resources as target")
	flag.StringVar(&flagExclude, "exclude", "", "TagName=Value,... If Tag exists exclude the resource")
	flag.BoolVar(&flagDryRun, "dryrun", false, "show process target only")
	flag.BoolVar(&flagVersion, "version", false, "display version")
	flag.Parse()

	if flagVersion {
		fmt.Printf("let-rds-sleep %s", version)
		os.Exit(0)
	}

	app, err := lrs.New(
		flagMode,
		lrs.OptTarget(flagTarget),
		lrs.OptExclude(flagExclude),
		lrs.OptDryRun(flagDryRun),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init app: %s", err)
	}

	if strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") || os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		fmt.Println("started as Lambda function handler")
		lambda.Start(app.HandleRequest)
	} else {
		fmt.Println("started as oneshot app")
		if err := app.Run(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "failed to Run: %s", err)
			os.Exit(1)
		}
	}

	fmt.Println("bye")
}
