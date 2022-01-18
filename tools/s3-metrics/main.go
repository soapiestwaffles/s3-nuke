package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/dustin/go-humanize"
	"github.com/guptarohit/asciigraph"
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

	bucketRegion, err := s3svc.GetBucektRegion(context.TODO(), selectedBucket)
	if err != nil {
		fmt.Println("Error detecting bucket region!", err)
		os.Exit(1)
	}
	fmt.Println("🌎 -> bucket located in", bucketRegion)
	fmt.Println("")

	cloudwatchSvc := cloudwatch.NewService(cloudwatch.WithAWSEndpoint(cli.AWSEndpoint), cloudwatch.WithRegion(bucketRegion))

	// loadingSpinner.Suffix = " fetching bucket metrics..."
	// loadingSpinner.Start()
	objectCountResults, err := cloudwatchSvc.GetS3ObjectCount(context.TODO(), selectedBucket, 720, 60)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	objInBucket := objectCountResults.Values[0]
	lastUpdate := objectCountResults.Timestamps[0]

	for i, j := 0, len(objectCountResults.Values)-1; i < j; i, j = i+1, j-1 {
		objectCountResults.Values[i], objectCountResults.Values[j] = objectCountResults.Values[j], objectCountResults.Values[i]
	}

	graph := asciigraph.Plot(objectCountResults.Values, asciigraph.Width(60), asciigraph.Height(10), asciigraph.Caption("Object Count for past 30 Days"))
	fmt.Println(graph)
	fmt.Println("")
	fmt.Println("Approx. objects currently in bucket:", humanize.Comma(int64(objInBucket)))
	fmt.Printf("Metric last updated: %s at %s\n", humanize.Time(lastUpdate.Local()), lastUpdate.Local())
}
