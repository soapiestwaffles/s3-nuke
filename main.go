package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"time"

	"github.com/Avalanche-io/counter"
	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/dustin/go-humanize"
	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/assets"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/ui/tui"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/workers"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/cloudwatch"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
	"golang.org/x/sync/errgroup"
)

const releaseURL = "https://github.com/soapiestwaffles/s3-nuke/releases"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	// builtBy = "unknown"

	cli struct {
		Version     bool   `help:"display version information" optional:""`
		AWSEndpoint string `help:"override AWS endpoint address" short:"e" optional:"" env:"AWS_ENDPOINT"`
		Concurrency int    `help:"amount of concurrency used during delete operations" optional:"" default:"100"`
		Debug       bool   `help:"enable debugging output (warning: this is very verbose)" optional:""`
	}
)

func main() {
	kongCtx := kong.Parse(&cli,
		kong.Name("s3-nuke"),
		kong.Description("Quickly destroy all objects and versions in an AWS S3 bucket."))

	if _, regionEnv := os.LookupEnv("AWS_REGION"); !regionEnv {
		os.Setenv("AWS_REGION", "us-east-1")
	}

	fmt.Println(assets.Logo)

	//  Show version information and exit
	if cli.Version {
		fmt.Println("Find new releases at", releaseURL)
		fmt.Println("")
		fmt.Println("version....:", version)
		fmt.Println("commit.....:", commit)
		fmt.Println("date.......:", date)
		os.Exit(0)
	}

	if cli.AWSEndpoint != "" {
		fmt.Println("Using AWS endpoint:", cli.AWSEndpoint)
	}

	ctx := context.Background()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false})
	if cli.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Info().Msg("debug logging output enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	// Set up S3 client
	var s3svc s3.Service
	s3svc = s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint))

	// Get list of buckets
	loadingSpinner := spinner.New(spinner.CharSets[13], 100*time.Millisecond)
	loadingSpinner.Suffix = " fetching bucket list..."
	err := loadingSpinner.Color("blue", "bold")
	kongCtx.FatalIfErrorf(err)
	if !cli.Debug {
		loadingSpinner.Start()
	}
	log.Debug().Msg("s3: get all buckets")
	buckets, err := s3svc.GetAllBuckets(ctx)
	kongCtx.FatalIfErrorf(err)
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
	if !cli.Debug {
		loadingSpinner.Start()
	}
	log.Debug().Msg("s3: get bucket region")
	bucketRegion, err := s3svc.GetBucektRegion(ctx, selectedBucket)
	loadingSpinner.Stop()
	if err != nil {
		fmt.Println("Error detecting bucket region!", err)
		os.Exit(1)
	}
	fmt.Println("🌎 bucket located in", bucketRegion)
	fmt.Println("")

	// recreate s3svc with bucket's region, create cloudwatch svc
	s3svc = s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint), s3.WithRegion(bucketRegion))
	cloudwatchSvc := cloudwatch.NewService(cloudwatch.WithAWSEndpoint(cli.AWSEndpoint), cloudwatch.WithRegion(bucketRegion))

	// Fetch bucket metrics
	loadingSpinner.Suffix = " fetching bucket metrics..."
	if !cli.Debug {
		loadingSpinner.Start()
	}
	log.Debug().Str("bucket", selectedBucket).Msg("fetching cloudwatch bucket metrics")
	objectCountResults, _ := cloudwatchSvc.GetS3ObjectCount(ctx, selectedBucket, 720, 60)
	loadingSpinner.Stop()

	if len(objectCountResults.Values) > 0 {
		fmt.Printf("Bucket object count: %s\n", humanize.Comma(int64(objectCountResults.Values[0])))
		fmt.Printf("(object count metric last updated %s @ %s)\n", humanize.Time(objectCountResults.Timestamps[0].Local()), objectCountResults.Timestamps[0].Local())
		fmt.Println("")
	} else {
		log.Debug().Str("bucket", selectedBucket).Msg("bucket metrics were not available")
	}

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

	println("")

	log.Debug().Str("bucket", selectedBucket).Int("concurrency", cli.Concurrency).Msg("starting nuke")
	err = nuke(ctx, s3svc, selectedBucket, cli.Concurrency)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

}

// Delete operation w/progress bar
func nuke(ctx context.Context, s3svc s3.Service, bucket string, concurrency int) error {
	fmt.Println("")

	c := counter.New()
	bar := progressbar.Default(-1, "deleting objects...")
	err := bar.RenderBlank()
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	s3DeleteQueue := make(chan s3.ObjectIdentifier, 100000)
	deleteProgress := make(chan int, concurrency*2)
	var progressWG sync.WaitGroup
	progressWG.Add(1)
	go func() {
		for progressUpdate := range deleteProgress {
			_ = bar.Add(progressUpdate)
		}
		progressWG.Done()
	}()

	g.Go(func() error {
		defer close(s3DeleteQueue)

		c, err := workers.S3QueueObjectVersions(ctx, s3svc, bucket, s3DeleteQueue)
		if err != nil {
			return err
		}
		log.Debug().Int("totalObjectsAddedToQueue", c).Msg("all objects added to queue")

		return nil
	})

	for i := 0; i < concurrency; i++ {
		g.Go(func() error {
			deleteCount, err := workers.S3DeleteFromChannel(ctx, s3svc, bucket, s3DeleteQueue, deleteProgress)
			if err != nil {
				return err
			}
			log.Debug().Int("objectsDeleted", deleteCount).Msg("worker finished")
			c.Add(int64(deleteCount))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	close(deleteProgress)
	progressWG.Wait()

	fmt.Println("")
	fmt.Println("")
	fmt.Println("💣  --- Nuke complete! ---  💣")
	fmt.Println("")
	fmt.Printf("Removed %s objects\n", humanize.Comma(c.Get()))

	return nil
}
