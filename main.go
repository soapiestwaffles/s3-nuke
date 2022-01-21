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
	"github.com/schollz/progressbar/v3"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/assets"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/ui/tui"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/cloudwatch"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
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

	// autodetect bucket region
	loadingSpinner.Suffix = " fetching bucket region..."
	loadingSpinner.Start()
	bucketRegion, err := s3svc.GetBucektRegion(context.TODO(), selectedBucket)
	loadingSpinner.Stop()
	if err != nil {
		fmt.Println("Error detecting bucket region!", err)
		os.Exit(1)
	}
	fmt.Println("🌎 -> bucket located in", bucketRegion)
	fmt.Println("")

	// recreate s3svc with bucket's region, create cloudwatch svc
	s3svc = s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint), s3.WithRegion(bucketRegion))
	cloudwatchSvc := cloudwatch.NewService(cloudwatch.WithAWSEndpoint(cli.AWSEndpoint), cloudwatch.WithRegion(bucketRegion))

	// Warning message
	fmt.Println("⚠️   !!! WARNING !!!  ⚠️")
	fmt.Println("This will destroy all versions of all objects in the selected bucket")
	fmt.Println("")

	// Confirmation 1
	if !tui.TypeMatchingPhrase() {
		fmt.Println("")
		fmt.Println("Phrase did not match. Exiting!")
		os.Exit(1)
	}

	// Confirmation 2
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("[bucket: %s] Are you sure, this operation cannot be undone", selectedBucket),
		IsConfirm: true,
	}
	result, err := prompt.Run()
	if err != nil || strings.ToLower(result) != "y" {
		fmt.Println("Command aborted!")
		return
	}

	// Fetch bucket metrics for progress bar
	loadingSpinner.Suffix = " fetching bucket metrics..."
	loadingSpinner.Start()
	objectCountResults, _ := cloudwatchSvc.GetS3ObjectCount(context.TODO(), selectedBucket, 720, 60)
	loadingSpinner.Stop()

	println("")

	var objectCount int64
	if len(objectCountResults.Values) > 0 {
		objectCount = int64(objectCountResults.Values[0])
	} else {
		objectCount = -1
	}

	nuke(s3svc, objectCount)

}

// Delete operation w/progress bar
func nuke(s3svc s3.Service, objectCount int64) {
	bar := progressbar.Default(objectCount, "deleting objects...")
	bar.RenderBlank()

	for i := 0; i < 1000; i++ {
		if i+1 == bar.GetMax() {
			bar.ChangeMax(bar.GetMax() + 1)
		}
		bar.Add(1)

		time.Sleep(time.Millisecond * 25)
	}

}
