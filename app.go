package lrs

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/hashicorp/logutils"
	"github.com/samber/lo"
)

type App struct {
	Mode        string
	TargetTags  []types.Tag // only resources having all of tags will be process
	ExcludeTags []types.Tag // resources having any of tags will not be process
	DryRun      bool
}

const (
	ModeStop  = "stop"
	ModeStart = "start"
)

type Processor func(ctx context.Context, svc *rds.Client, target Resource) error

type Resource struct {
	ID     string
	Type   string // instance or cluster
	Status string
	Tags   []types.Tag
}

func (r Resource) String() string {
	return fmt.Sprintf("%s/%s", r.Type, r.ID)
}

func (r Resource) TagsAsString() string {
	l := []string{}

	for _, tag := range r.Tags {
		l = append(l, fmt.Sprintf("%s=%s", *tag.Key, *tag.Value))
	}

	return strings.Join(l, ",")
}

type Option func(app *App) error

func init() {
	logLevel := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "INFO"
	}

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(logLevel),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
}

func OptTarget(tagText string) Option {
	return func(app *App) error {
		tags, err := parseTags(tagText)
		if err != nil {
			return fmt.Errorf("failed to parse tagText: `%s`, err: %w", tagText, err)
		}

		app.TargetTags = tags

		return nil
	}
}

func OptExclude(tagText string) Option {
	return func(app *App) error {
		tags, err := parseTags(tagText)
		if err != nil {
			return fmt.Errorf("failed to parse tagText: `%s`, err: %w", tagText, err)
		}

		app.ExcludeTags = tags

		return nil
	}
}

func OptDryRun(dryrun bool) Option {
	return func(app *App) error {
		app.DryRun = dryrun
		return nil
	}
}

func New(options ...Option) (*App, error) {
	app := App{}

	for _, opt := range options {
		if err := opt(&app); err != nil {
			return nil, fmt.Errorf("failed to apply option:%+v, err: %w", opt, err)
		}
	}

	return &app, nil
}

func parseTags(text string) ([]types.Tag, error) {
	if text == "" {
		return []types.Tag{}, nil
	}
	tags := []types.Tag{}

	for _, tagText := range strings.Split(text, ",") {
		values := strings.SplitN(tagText, "=", 2)

		if len(values) != 2 {
			return nil, fmt.Errorf("invalid tag text: %s", text)
		}

		tags = append(tags, types.Tag{Key: aws.String(values[0]), Value: aws.String(values[1])})
	}

	return tags, nil
}

type LambdaEvent struct {
	Mode string `json:"mode"`
}

func (app *App) HandleRequest(ctx context.Context, event *LambdaEvent) error {
	return app.Run(ctx, event.Mode)
}

func (app *App) Run(ctx context.Context, mode string) error {
	log.Printf("[INFO] running to %s targets", mode)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load aws config: %w", err)
	}

	svc := rds.NewFromConfig(cfg)

	targets, err := app.listTargetResouces(ctx, svc)
	if err != nil {
		log.Printf("[DEBUG] failed to listTargetResources")
		return err
	}

	processor, err := app.selectProcessor(mode)
	if err != nil {
		return fmt.Errorf("[ERROR] failed to select processor: %w", err)
	}

	for _, target := range targets {
		log.Printf("[INFO] processing %s", target)

		if app.DryRun {
			log.Printf("[INFO] %s will be %s [dryrun]", target, mode)
			continue
		}

		err := processor(ctx, svc, target)
		if err != nil {
			log.Printf("[DEBUG] failed to process target: %s", target)
			return err
		}

		log.Printf("[INFO] process completed for %s", target)
	}

	return nil
}

func (app *App) listTargetResouces(ctx context.Context, svc *rds.Client) ([]Resource, error) {
	resources := []Resource{}

	{
		out, err := svc.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{})
		if err != nil {
			log.Printf("[DEBUG] failed to DescribeDBClusters")
			return nil, err
		}

		for _, c := range out.DBClusters {
			resources = append(resources, Resource{
				ID:     *c.DBClusterIdentifier,
				Type:   "cluster",
				Status: *c.Status,
				Tags:   c.TagList,
			})
		}
	}

	{
		out, err := svc.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
		if err != nil {
			log.Printf("[DEBUG] failed to DescribeDBInstances")
			return nil, err
		}

		for _, d := range out.DBInstances {
			if d.DBClusterIdentifier != nil {
				continue
			}

			resources = append(resources, Resource{
				ID:     *d.DBInstanceIdentifier,
				Type:   "instance",
				Status: *d.DBInstanceStatus,
				Tags:   d.TagList,
			})
		}
	}

	return app.filterResources(resources), nil
}

func (app *App) filterResources(resources []Resource) []Resource {
	log.Printf("[DEBUG] checking resource(s) satisfy target condition(s)")

	candidates := []Resource{}

	for _, r := range resources {
		log.Printf("[DEBUG] checking resource %s", r)
		satisfied := true

		for _, tag := range app.TargetTags {
			_, ok := lo.Find(r.Tags, func(t types.Tag) bool {
				return *tag.Key == *t.Key && *tag.Value == *t.Value
			})

			if !ok {
				satisfied = false
				log.Printf("[DEBUG] %s is not a target because tags %s are not satisfy condition(s)", r, r.TagsAsString())
				break
			}
		}

		if satisfied {
			candidates = append(candidates, r)
			log.Printf("[DEBUG] %s satisfies target condition, then added to candidates", r)
		}
	}

	log.Printf("[DEBUG] checking resource(s) match exclude condition(s)")

	filteredResources := []Resource{}

	for _, c := range candidates {
		exclude := false
		for _, tag := range app.ExcludeTags {
			_, ok := lo.Find(c.Tags, func(t types.Tag) bool {
				return *tag.Key == *t.Key && *tag.Value == *t.Value
			})

			if ok {
				exclude = true
				log.Printf("[DEBUG] %s removed from candidates because it has a tag %s=%s", c, *tag.Key, *tag.Value)
				break
			}
		}

		if !exclude {
			filteredResources = append(filteredResources, c)
			log.Printf("[DEBUG] %s not matches exclude condition(s), then added to target resources", c)
		}
	}

	return filteredResources
}

func start(ctx context.Context, svc *rds.Client, target Resource) error {
	if target.Status != "stopped" {
		log.Printf("[DEBUG] target is not stopped. nothing to do")
		return nil
	}

	switch target.Type {
	case "instance":
		_, err := svc.StartDBInstance(ctx, &rds.StartDBInstanceInput{
			DBInstanceIdentifier: aws.String(target.ID),
		})

		if err != nil {
			log.Printf("[DEBUG] failed to StartDBInstance target: %s", target)
			return err
		}
	case "cluster":
		_, err := svc.StartDBCluster(ctx, &rds.StartDBClusterInput{
			DBClusterIdentifier: aws.String(target.ID),
		})

		if err != nil {
			log.Printf("[DEBUG] failed to StartDBCluster target: %s", target)
			return err
		}
	default:
		log.Printf("[WARN] unknown target type: %s", target.Type)
		return nil
	}

	log.Printf("[INFO] successfully requested to start %s", target)

	return nil
}

func stop(ctx context.Context, svc *rds.Client, target Resource) error {
	if target.Status != "available" {
		log.Printf("[DEBUG] target is not available. nothing to do")
		return nil
	}

	switch target.Type {
	case "instance":
		_, err := svc.StopDBInstance(ctx, &rds.StopDBInstanceInput{
			DBInstanceIdentifier: aws.String(target.ID),
		})

		if err != nil {
			log.Printf("[DEBUG] failed to StopDBInstance target: %s", target)
			return err
		}
	case "cluster":
		_, err := svc.StopDBCluster(ctx, &rds.StopDBClusterInput{
			DBClusterIdentifier: aws.String(target.ID),
		})

		if err != nil {
			log.Printf("[DEBUG] failed to StopDBCluster target: %s", target)
			return err
		}

	default:
		log.Printf("[WARN] unknown target type: %s", target.Type)
		return nil
	}

	log.Printf("[INFO] successfully requested to stop %s", target)

	return nil
}

func (app *App) selectProcessor(mode string) (Processor, error) {
	switch mode {
	case ModeStart:
		return start, nil
	case ModeStop:
		return stop, nil
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}
