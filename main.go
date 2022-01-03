package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"time"

	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/manifoldco/promptui"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/assets"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws"
)

const releaseURL = "https://github.com/soapiestwaffles/s3-nuke/release"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	// builtBy = "unknown"

	cli struct {
		// Debug       bool   `help:"enable debug mode"`
		Version     bool   `help:"display version information" optional:""`
		AWSEndpoint string `help:"override AWS endpoint address" short:"e" optional:"" env:"AWS_ENDPOINT"`
		Region      string `help:"override AWS region" optional:"" env:"AWS_REGION"`
	}
)

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("s3-nuke"),
		kong.Description("Quickly destroy all objects and versions in an AWS S3 bucket."))

	fmt.Println(assets.Logo)

	//  Show version information and exit
	if cli.Version {
		fmt.Println("Find releases at", releaseURL)
		fmt.Println("")
		fmt.Println("version....:", version)
		fmt.Println("commit.....:", commit)
		fmt.Println("data.......:", date)
		os.Exit(0)
	}

	if cli.AWSEndpoint != "" {
		fmt.Println("Using AWS endpoint:", cli.AWSEndpoint)
	}
	// Set up S3 client
	var s3 aws.S3Service
	if cli.Region == "" {
		s3 = aws.NewS3Service(aws.WithAWSEndpoint(cli.AWSEndpoint), aws.WithRegion(cli.Region))
	} else {
		s3 = aws.NewS3Service(aws.WithAWSEndpoint(cli.AWSEndpoint))
	}

	// Get list of buckets
	spinnerGetBuckets := spinner.New(spinner.CharSets[13], 100*time.Millisecond)
	spinnerGetBuckets.Suffix = " fetching bucket list..."
	err := spinnerGetBuckets.Color("blue", "bold")
	ctx.FatalIfErrorf(err)
	spinnerGetBuckets.Start()
	buckets, err := s3.GetAllBuckets(context.TODO())
	ctx.FatalIfErrorf(err)
	spinnerGetBuckets.Stop()

	// Exit if there are no buckets to nuke
	if len(buckets) == 0 {
		fmt.Println("No buckets found! Exiting.")
		os.Exit(0)
	}

	// User select bucket
	selectBucketsPrompt(buckets)

	ctx.FatalIfErrorf(err)

}

func selectBucketsPrompt(buckets []aws.Bucket) {
	// This is a nasty hack just to dereference the `Name` field.
	// TODO investigate more to see if we can dereference right in the template OR find a different UI library
	derefBucket := []struct {
		Name         string
		CreationDate *time.Time
	}{}
	for _, b := range buckets {
		derefBucket = append(derefBucket, struct {
			Name         string
			CreationDate *time.Time
		}{*b.Name, b.CreationDate})
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ \"---\" | faint }} {{ . | blue | bold }} {{ \"---\" | faint }}",
		Active:   "\U0001FAA3  {{ .Name | cyan }}",
		Inactive: "   {{ .Name | cyan }}",
		Selected: "\U0001FAA3  {{ .Name | bold | green }}",
		Details: `
------ S3 Bucket Info ------
{{ "Name............:" | faint }} {{ .Name }}
{{ "Creation Date...:" | faint }} {{ .CreationDate }}`,
	}

	searcher := func(input string, index int) bool {
		bucket := derefBucket[index]
		name := strings.ReplaceAll(strings.ToLower(bucket.Name), " ", "")
		input = strings.ReplaceAll(strings.ToLower(input), " ", "")

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     "Buckets to Nuke",
		Items:     derefBucket,
		Templates: templates,
		Size:      5,
		Searcher:  searcher,
	}

	_, _, err := prompt.Run()

	if err != nil {
		return
	}

}
