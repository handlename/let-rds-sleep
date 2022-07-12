package lrs

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseTags(t *testing.T) {
	cases := []struct {
		title string
		input string
		want  []types.Tag
		err   error
	}{
		{
			title: "no tags",
			input: "",
			want:  []types.Tag{},
		},
		{
			title: "1 tag",
			input: "Foo=Bar",
			want: []types.Tag{
				{Key: aws.String("Foo"), Value: aws.String("Bar")},
			},
		},
		{
			title: "3 tags",
			input: "Hoge=1,Fuga=2,Piyo=3",
			want: []types.Tag{
				{Key: aws.String("Hoge"), Value: aws.String("1")},
				{Key: aws.String("Fuga"), Value: aws.String("2")},
				{Key: aws.String("Piyo"), Value: aws.String("3")},
			},
		},
		{
			title: "invalid text",
			input: "somethingwrong",
			want:  nil,
			err:   fmt.Errorf("invalid tag text: somethingwrong"),
		},
	}

	for _, c := range cases {
		got, err := parseTags(c.input)
		if err != nil && err.Error() != c.err.Error() {
			t.Errorf("unexpected error: got `%s`, want `%s`", err, c.err)
		}
		if err == nil && err != c.err {
			t.Errorf("expected err but got nil")
		}

		if diff := cmp.Diff(c.want, got, cmpopts.IgnoreUnexported(types.Tag{})); diff != "" {
			t.Errorf("unexpected tags:\n%s", diff)
		}
	}
}

func TestAppFilterResources(t *testing.T) {
	rFooBar := Resource{
		ID: "Foo1Bar2",
		Tags: []types.Tag{
			{Key: aws.String("Foo"), Value: aws.String("1")},
			{Key: aws.String("Bar"), Value: aws.String("2")},
		},
	}
	rFooBuzz := Resource{
		ID: "FooBuzz",
		Tags: []types.Tag{
			{Key: aws.String("Foo"), Value: aws.String("1")},
			{Key: aws.String("Buzz"), Value: aws.String("3")},
		},
	}
	rBarBuzz := Resource{
		ID: "BarBuzz",
		Tags: []types.Tag{
			{Key: aws.String("Bar"), Value: aws.String("2")},
			{Key: aws.String("Buzz"), Value: aws.String("3")},
		},
	}

	cases := []struct {
		title         string
		inResources   []Resource
		inTargetTags  []types.Tag
		inExcludeTags []types.Tag
		want          []Resource
	}{
		{
			title:         "no condition",
			inResources:   []Resource{rFooBar, rFooBuzz, rBarBuzz},
			inTargetTags:  []types.Tag{},
			inExcludeTags: []types.Tag{},
			want:          []Resource{rFooBar, rFooBuzz, rBarBuzz},
		},
		{
			title:       "set target Foo=1,Bar=2 only",
			inResources: []Resource{rFooBar, rFooBuzz, rBarBuzz},
			inTargetTags: []types.Tag{
				{Key: aws.String("Foo"), Value: aws.String("1")},
				{Key: aws.String("Bar"), Value: aws.String("2")},
			},
			inExcludeTags: []types.Tag{},
			want:          []Resource{rFooBar},
		},
		{
			title:        "set exclude Foo=1 only",
			inResources:  []Resource{rFooBar, rFooBuzz, rBarBuzz},
			inTargetTags: []types.Tag{},
			inExcludeTags: []types.Tag{
				{Key: aws.String("Foo"), Value: aws.String("1")},
			},
			want: []Resource{rBarBuzz},
		},
		{
			title:       "set target Foo=1 and exclude Bar=2",
			inResources: []Resource{rFooBar, rFooBuzz, rBarBuzz},
			inTargetTags: []types.Tag{
				{Key: aws.String("Foo"), Value: aws.String("1")},
			},
			inExcludeTags: []types.Tag{
				{Key: aws.String("Bar"), Value: aws.String("2")},
			},
			want: []Resource{rFooBuzz},
		},
		{
			title:       "target value not match",
			inResources: []Resource{rFooBar, rFooBuzz, rBarBuzz},
			inTargetTags: []types.Tag{
				{Key: aws.String("Foo"), Value: aws.String("9")},
			},
			inExcludeTags: []types.Tag{},
			want:          []Resource{},
		},
	}

	for _, c := range cases {
		t.Logf("testing `%s`", c.title)

		app := App{
			TargetTags:  c.inTargetTags,
			ExcludeTags: c.inExcludeTags,
		}

		got := app.filterResources(c.inResources)

		if diff := cmp.Diff(c.want, got, cmpopts.IgnoreUnexported(types.Tag{})); diff != "" {
			t.Errorf("unexpected resources:\n%s", diff)
		}
	}
}
