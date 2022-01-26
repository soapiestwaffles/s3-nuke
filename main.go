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
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/cloudwatch"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
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
		Region      string `help:"override AWS region" optional:"" default:"us-east-1" env:"AWS_REGION"`
		Concurrency int    `help:"amount of concurrency used during delete operations" optional:"" default:"100"`
		Debug       bool   `help:"enable debugging output (warning: this is very verbose)" optional:""`
	}
)

func main() {
	kongCtx := kong.Parse(&cli,
		kong.Name("s3-nuke"),
		kong.Description("Quickly destroy all objects and versions in an AWS S3 bucket."))

	if _, regionEnv := os.LookupEnv("AWS_REGION"); !regionEnv {
		os.Setenv("AWS_REGION", cli.Region)
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
	if cli.Region == "" {
		s3svc = s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint), s3.WithRegion(cli.Region))
	} else {
		s3svc = s3.NewService(s3.WithAWSEndpoint(cli.AWSEndpoint))
	}

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

	var wg sync.WaitGroup

	s3ListQueue := make(chan string, 100000)
	s3VersionListQueue := make(chan s3.ObjectVersion, 100000)
	s3ListQueueLogger := log.With().Str("queue", "s3ListQueue").Logger()
	s3VersionListQueueLogger := log.With().Str("queue", "s3VersionListQueue").Logger()
	// Get top level objects
	wg.Add(1)
	go func() {
		var continuationToken *string
		for {
			s3ListQueueLogger.Debug().Interface("continuationToken", continuationToken).Msg("s3: list objects")
			objects, token, err := s3svc.ListObjects(ctx, bucket, continuationToken, nil)
			if err != nil {
				s3ListQueueLogger.Fatal().Err(err).Msg("failed to list objects")
				os.Exit(1)
			}

			for _, object := range objects {
				s3ListQueueLogger.Debug().Str("object", object).Msg("add object to queue")
				s3ListQueue <- object
			}

			if token == nil {
				break
			}
			continuationToken = token
		}
		close(s3ListQueue)
		wg.Done()
	}()

	// Begin nuke operations: primary objects
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(input chan string) {
			deleteQueue := []s3.ObjectIdentifier{}
			for key := range input {
				deleteQueue = append(deleteQueue, s3.ObjectIdentifier{
					Key: &key,
				})

				// flush
				if len(deleteQueue) == 1000 {
					s3ListQueueLogger.Debug().Msg("1000 flush queue")
					deleteResult, err := s3svc.DeleteObjects(ctx, bucket, deleteQueue)
					s3ListQueueLogger.Debug().Int("objectCount", len(deleteResult)).Msg("deleted objects")
					if err != nil {
						s3ListQueueLogger.Fatal().Err(err).Msg("failed to delete objects")
						os.Exit(1)
					}
					_ = bar.Add(1000)
					c.Add(1000)

					// reset
					deleteQueue = []s3.ObjectIdentifier{}
				}
			}

			// final flush
			if len(deleteQueue) > 0 {
				s3ListQueueLogger.Debug().Int("remainingQueueDepth", len(deleteQueue)).Msg("final flush queue")
				deleteResult, err := s3svc.DeleteObjects(ctx, bucket, deleteQueue)
				s3ListQueueLogger.Debug().Int("objectCount", len(deleteResult)).Msg("deleted objects")
				if err != nil {
					s3ListQueueLogger.Fatal().Err(err).Msg("failed to delete objects")
					os.Exit(1)
				}
				_ = bar.Add(len(deleteQueue))
				c.Add(int64(len(deleteQueue)))
			}

			wg.Done()
		}(s3ListQueue)
	}

	wg.Wait()

	// Get object versions
	wg.Add(1)
	go func() {
		var keyMarkerToken, versionMarkerToken *string
		for {
			s3VersionListQueueLogger.Debug().Str("bucket", bucket).
				Interface("keyMarkerToken", keyMarkerToken).
				Interface("versionMarkerToken", versionMarkerToken).
				Msg("s3: list objects versions")
			objectVersions, keyMarker, versionMarker, err := s3svc.ListObjectVersions(ctx, bucket, keyMarkerToken, versionMarkerToken, nil)
			if err != nil {
				s3VersionListQueueLogger.Fatal().Err(err).Msg("failed to list object versions")
				os.Exit(1)
			}

			for _, version := range objectVersions {
				s3VersionListQueueLogger.Debug().Str("object", *version.Key).Str("versionID", *version.VersionID).Msg("add object version to queue")
				s3VersionListQueue <- version
			}

			if keyMarker == nil && versionMarker == nil {
				break
			}
			keyMarkerToken = keyMarker
			versionMarkerToken = versionMarker
		}
		close(s3VersionListQueue)
		wg.Done()
	}()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(input chan s3.ObjectVersion) {
			deleteQueue := []s3.ObjectIdentifier{}
			for key := range input {
				deleteQueue = append(deleteQueue, s3.ObjectIdentifier{
					Key:       key.ObjectIdentifier.Key,
					VersionID: key.ObjectIdentifier.VersionID,
				})

				// flush
				if len(deleteQueue) == 1000 {
					s3VersionListQueueLogger.Debug().Msg("1000 flush queue")
					deleteResult, err := s3svc.DeleteObjects(ctx, bucket, deleteQueue)
					s3VersionListQueueLogger.Debug().Int("objectCount", len(deleteResult)).Msg("deleted objects")
					if err != nil {
						s3VersionListQueueLogger.Fatal().Err(err).Msg("failed to delete objects")
						os.Exit(1)
					}
					_ = bar.Add(1000)
					c.Add(1000)
					// reset
					deleteQueue = []s3.ObjectIdentifier{}
				}
			}

			// final flush
			if len(deleteQueue) > 0 {
				s3VersionListQueueLogger.Debug().Int("remainingQueueDepth", len(deleteQueue)).Msg("final flush queue")
				deleteResult, err := s3svc.DeleteObjects(ctx, bucket, deleteQueue)
				s3VersionListQueueLogger.Debug().Int("objectCount", len(deleteResult)).Msg("deleted objects")
				if err != nil {
					s3VersionListQueueLogger.Fatal().Err(err).Msg("failed to delete objects")
					os.Exit(1)
				}
				_ = bar.Add(len(deleteQueue))
				c.Add(int64(len(deleteQueue)))
			}

			wg.Done()
		}(s3VersionListQueue)
	}

	wg.Wait()

	fmt.Println("")
	fmt.Println("")
	fmt.Println("💣  --- Nuke complete! ---  💣")
	fmt.Println("")
	fmt.Printf("Removed %s objects\n", humanize.Comma(c.Get()))

	return nil
}
