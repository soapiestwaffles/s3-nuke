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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/ui/tui"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/cloudwatch"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

const releaseURL = "https://github.com/soapiestwaffles/s3-nuke/releases"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	cli struct {
		Version     bool   `help:"display version information" optional:""`
		AWSEndpoint string `help:"override AWS endpoint address" short:"e" optional:"" env:"AWS_ENDPOINT"`
		Debug       bool   `help:"enable debugging output" optional:""`
	}
)

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("s3-metrics"),
		kong.Description("s3-metrics tool: get bucket object metrics for a particular bucket"))

	//  Show version information and exit
	if cli.Version {
		fmt.Println("version....:", version)
		fmt.Println("commit.....:", commit)
		fmt.Println("date.......:", date)
		fmt.Println("")
		fmt.Println("Find new releases at", releaseURL)
		fmt.Println("")
		os.Exit(0)
	}

	if _, regionEnv := os.LookupEnv("AWS_REGION"); !regionEnv {
		if err := os.Setenv("AWS_REGION", "us-east-1"); err != nil {
			log.Warn().Err(err).Msg("Failed to set AWS_REGION environment variable")
		}
	}

	if cli.AWSEndpoint != "" {
		fmt.Println("Using AWS endpoint:", cli.AWSEndpoint)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false})
	if cli.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Info().Msg("debug logging output enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	// Set up S3 client
	s3svc := s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint))

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

	bucketRegion, err := s3svc.GetBucketRegion(context.TODO(), selectedBucket)
	if err != nil {
		fmt.Println("Error detecting bucket region!", err)
		os.Exit(1)
	}
	fmt.Println("🌎 -> bucket located in", bucketRegion)
	fmt.Println("")

	cloudwatchSvc := cloudwatch.NewService(cloudwatch.WithAWSEndpoint(cli.AWSEndpoint), cloudwatch.WithRegion(bucketRegion))

	loadingSpinner.Suffix = " fetching bucket metrics..."
	loadingSpinner.Start()
	objectCountResults, err := cloudwatchSvc.GetS3ObjectCount(context.TODO(), selectedBucket, 720, 60)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	byteCountResults, err := cloudwatchSvc.GetS3ByteCount(context.TODO(), selectedBucket, cloudwatch.StandardStorage, 720, 60)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	loadingSpinner.Stop()

	if len(objectCountResults.Values) == 0 || len(byteCountResults.Values) == 0 {
		fmt.Println("")
		fmt.Println("no cloudwatch metrics found for bucket!")
		os.Exit(0)
	}

	objInBucket := objectCountResults.Values[0]
	objLastUpdate := objectCountResults.Timestamps[0]

	bytesInBucket := byteCountResults.Values[0]
	bytesLastUpdate := byteCountResults.Timestamps[0]

	// ⠐⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠂
	// ⠄⠄⣰⣾⣿⣿⣿⠿⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣆⠄⠄
	// ⠄⠄⣿⣿⣿⡿⠋⠄⡀⣿⣿⣿⣿⣿⣿⣿⣿⠿⠛⠋⣉⣉⣉⡉⠙⠻⣿⣿⠄⠄
	// ⠄⠄⣿⣿⣿⣇⠔⠈⣿⣿⣿⣿⣿⡿⠛⢉⣤⣶⣾⣿⣿⣿⣿⣿⣿⣦⡀⠹⠄⠄
	// ⠄⠄⣿⣿⠃⠄⢠⣾⣿⣿⣿⠟⢁⣠⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡄⠄⠄
	// ⠄⠄⣿⣿⣿⣿⣿⣿⣿⠟⢁⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⠄⠄
	// ⠄⠄⣿⣿⣿⣿⣿⡟⠁⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠄⠄
	// ⠄⠄⣿⣿⣿⣿⠋⢠⣾⣿⣿⣿⣿⣿⣿⡿⠿⠿⠿⠿⣿⣿⣿⣿⣿⣿⣿⣿⠄⠄
	// ⠄⠄⣿⣿⡿⠁⣰⣿⣿⣿⣿⣿⣿⣿⣿⠗⠄⠄⠄⠄⣿⣿⣿⣿⣿⣿⣿⡟⠄⠄
	// ⠄⠄⣿⡿⠁⣼⣿⣿⣿⣿⣿⣿⡿⠋⠄⠄⠄⣠⣄⢰⣿⣿⣿⣿⣿⣿⣿⠃⠄⠄
	// ⠄⠄⡿⠁⣼⣿⣿⣿⣿⣿⣿⣿⡇⠄⢀⡴⠚⢿⣿⣿⣿⣿⣿⣿⣿⣿⡏⢠⠄⠄
	// ⠄⠄⠃⢰⣿⣿⣿⣿⣿⣿⡿⣿⣿⠴⠋⠄⠄⢸⣿⣿⣿⣿⣿⣿⣿⡟⢀⣾⠄⠄
	// ⠄⠄⢀⣿⣿⣿⣿⣿⣿⣿⠃⠈⠁⠄⠄⢀⣴⣿⣿⣿⣿⣿⣿⣿⡟⢀⣾⣿⠄⠄
	// ⠄⠄⢸⣿⣿⣿⣿⣿⣿⣿⠄⠄⠄⠄⢶⣿⣿⣿⣿⣿⣿⣿⣿⠏⢀⣾⣿⣿⠄⠄
	// ⠄⠄⣿⣿⣿⣿⣿⣿⣿⣷⣶⣶⣶⣶⣶⣿⣿⣿⣿⣿⣿⣿⠋⣠⣿⣿⣿⣿⠄⠄
	// ⠄⠄⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⢁⣼⣿⣿⣿⣿⣿⠄⠄
	// ⠄⠄⢻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⢁⣴⣿⣿⣿⣿⣿⣿⣿⠄⠄
	// ⠄⠄⠈⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⢁⣴⣿⣿⣿⣿⠗⠄⠄⣿⣿⠄⠄
	// ⠄⠄⣆⠈⠻⢿⣿⣿⣿⣿⣿⣿⠿⠛⣉⣤⣾⣿⣿⣿⣿⣿⣇⠠⠺⣷⣿⣿⠄⠄
	// ⠄⠄⣿⣿⣦⣄⣈⣉⣉⣉⣡⣤⣶⣿⣿⣿⣿⣿⣿⣿⣿⠉⠁⣀⣼⣿⣿⣿⠄⠄
	// ⠄⠄⠻⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣶⣾⣿⣿⡿⠟⠄⠄
	// ⠠⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄⠄
	for i, j := 0, len(objectCountResults.Values)-1; i < j; i, j = i+1, j-1 {
		objectCountResults.Values[i], objectCountResults.Values[j] = objectCountResults.Values[j], objectCountResults.Values[i]
	}

	for i, j := 0, len(byteCountResults.Values)-1; i < j; i, j = i+1, j-1 {
		byteCountResults.Values[i], byteCountResults.Values[j] = byteCountResults.Values[j], byteCountResults.Values[i]
	}

	byteGraph := asciigraph.Plot(byteCountResults.Values, asciigraph.Width(60), asciigraph.Height(10), asciigraph.Caption("Byte Count for past 30 Days (Standard Storage)"))
	fmt.Println(byteGraph)
	fmt.Println("")
	fmt.Println("Approx. bytes currently in bucket (standard storage):", humanize.Bytes(uint64(bytesInBucket)))
	fmt.Printf("Metric last updated: %s at %s\n", humanize.Time(bytesLastUpdate.Local()), bytesLastUpdate.Local())

	fmt.Println("")
	fmt.Println("")
	fmt.Println("")

	objGraph := asciigraph.Plot(objectCountResults.Values, asciigraph.Width(60), asciigraph.Height(10), asciigraph.Caption("Object Count for past 30 Days"))
	fmt.Println(objGraph)
	fmt.Println("")
	fmt.Println("Approx. objects currently in bucket:", humanize.Comma(int64(objInBucket)))
	fmt.Printf("Metric last updated: %s at %s\n", humanize.Time(objLastUpdate.Local()), objLastUpdate.Local())
}
