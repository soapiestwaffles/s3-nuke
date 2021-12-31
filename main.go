package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"time"

	"github.com/alecthomas/kong"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
		AWSEndpoint string `help:"override AWS endpoint address" short:"e" optional:""`
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

	// Initialize AWS S3 Client
	cfg, err := newAWSSDKConfig(cli.AWSEndpoint)
	ctx.FatalIfErrorf(err)
	s3Client := s3.NewFromConfig(cfg)

	// Init S3 Service
	s3 := aws.NewS3Service(s3Client)
	if s3 == nil {
		ctx.FatalIfErrorf(errors.New("error initializing s3 service"))
	}

	// Get list of buckets
	spinnerGetBuckets := spinner.New(spinner.CharSets[13], 100*time.Millisecond)
	spinnerGetBuckets.Suffix = " fetching bucket list..."
	err = spinnerGetBuckets.Color("blue", "bold")
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
		name := strings.Replace(strings.ToLower(bucket.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

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

func newAWSSDKConfig(awsEndpoint string) (awssdk.Config, error) {
	var cfg awssdk.Config
	var err error
	if cli.AWSEndpoint != "" {
		customResolver := awssdk.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (awssdk.Endpoint, error) {
			return awssdk.Endpoint{
				PartitionID: "aws",
				URL:         cli.AWSEndpoint,
			}, nil
		})
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolverWithOptions(customResolver))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	return cfg, err
}
