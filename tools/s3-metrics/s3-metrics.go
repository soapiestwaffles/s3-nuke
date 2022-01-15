package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/ui/tui"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/cloudwatch"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

var (
	cli struct {
		AWSEndpoint string `help:"override AWS endpoint address" short:"e" optional:"" env:"AWS_ENDPOINT"`
		Region      string `help:"specify region to create bucket and objects in" short:"r" default:"us-west-2" env:"AWS_REGION"`
	}
)

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("s3-metrics"),
		kong.Description("s3-metrics tool: get bucket object metrics for a particular bucket"))

	if cli.AWSEndpoint != "" {
		fmt.Println("Using AWS endpoint:", cli.AWSEndpoint)
	}

	var cloudwatchSvc cloudwatch.Service
	if cli.Region == "" {
		cloudwatchSvc = cloudwatch.NewService(cloudwatch.WithAWSEndpoint(cli.AWSEndpoint), cloudwatch.WithRegion(cli.Region))
	} else {
		cloudwatchSvc = cloudwatch.NewService(cloudwatch.WithAWSEndpoint(cli.AWSEndpoint))
	}

	// Set up S3 client
	var s3svc s3.Service
	if cli.Region == "" {
		s3svc = s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint), s3.WithRegion(cli.Region))
	} else {
		s3svc = s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint))
	}

	// Get list of buckets
	loadingSpinner := spinner.New(spinner.CharSets[13], 100*time.Millisecond)
	loadingSpinner.Suffix = " fetching bucket list..."
	err := loadingSpinner.Color("blue", "bold")
	ctx.FatalIfErrorf(err)
	loadingSpinner.Start()
	buckets, err := s3svc.GetAllBuckets(context.TODO())
	ctx.FatalIfErrorf(err)
	loadingSpinner.Stop()

	// Exit if there are no buckets to nuke
	if len(buckets) == 0 {
		fmt.Println("No buckets found! Exiting.")
		os.Exit(0)
	}

	// User select bucket
	fmt.Println("")
	selectedBucket, err := tui.SelectBucketsPrompt(buckets)
	if err != nil {
		fmt.Println("Error selecting bucket! Exiting.")
		os.Exit(1)
	}

	fmt.Println("")
	// loadingSpinner.Suffix = " fetching bucket metrics..."
	// loadingSpinner.Start()
	err = cloudwatchSvc.GetS3ObjectCount(context.TODO(), selectedBucket, 336, 60)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

}
