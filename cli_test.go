package lrs

import (
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseCLIFlags(t *testing.T) {
	tests := []struct {
		title string
		args  []string
		envs  map[string]string
		want  *CLIFlags
	}{
		{
			title: "default",
			args: []string{
				"-mode", "stop",
			},
			want: &CLIFlags{
				Mode:    "stop",
				Target:  "",
				Exclude: "",
				DryRun:  false,
			},
		},
		{
			title: "args only",
			args: []string{
				"-mode", "stop",
				"-target", "tag:Name=foo",
				"-exclude", "tag:Name=bar",
				"-dryrun",
			},
			want: &CLIFlags{
				Mode:    "stop",
				Target:  "tag:Name=foo",
				Exclude: "tag:Name=bar",
				DryRun:  true,
			},
		},
		{
			title: "args override envs",
			args: []string{
				"-mode", "stop",
				"-target", "tag:Name=foo",
			},
			envs: map[string]string{
				"LET_RDS_SLEEP_MODE":    "start",
				"LET_RDS_SLEEP_EXCLUDE": "tag:Name=bar",
			},
			want: &CLIFlags{
				Mode:    "stop",
				Target:  "tag:Name=foo",
				Exclude: "tag:Name=bar",
				DryRun:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet("let-rds-sleep", flag.ExitOnError)

			if tt.envs != nil {
				for k, v := range tt.envs {
					t.Setenv(k, v)
				}
			}

			got, err := parseCLIFlags(tt.args)
			if err != nil {
				t.Errorf("unexpected error: %s", err)

			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected result: %s", diff)
			}
		})
	}
}
