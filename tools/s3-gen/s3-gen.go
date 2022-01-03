package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/google/uuid"
	"github.com/gosuri/uiprogress"
	"github.com/manifoldco/promptui"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/generators"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws"
)

var (
	cli struct {
		AWSEndpoint  string `help:"override AWS endpoint address" short:"e" optional:"" env:"AWS_ENDPOINT"`
		NumBuckets   int    `help:"number of buckets with randomized names to create" short:"n" required:""`
		NumObjects   int    `help:"number of random objects generated and put into buckets" short:"o" required:""`
		NumVersions  int    `help:"number of versions to create for each random object" short:"v" required:""`
		Region       string `help:"specify region to create bucket and objects in" short:"r" default:"us-west-2"`
		BucketPrefix string `help:"prefix for every bucket name generated" short:"p" default:"s3gen" optional:""`
		YesFlag      bool   `name:"yes" help:"bypass user prompt and proceed with action automatically" optional:""`
	}
)

func main() {
	kong.Parse(&cli,
		kong.Name("s3-gen"),
		kong.Description("s3-nuke tool: generate a set of randomized buckets each containing a set of randomized objects and versions"))

	fmt.Println("=== RANDOM BUCKET GENERATOR ===")
	fmt.Println("")

	if cli.AWSEndpoint != "" {
		fmt.Println("Using AWS endpoint:", cli.AWSEndpoint)
		fmt.Println("")
	}

	if !cli.YesFlag {
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Create resources [%d bucket(s)]/[%d object(s)]/[%d version(s)]", cli.NumBuckets, cli.NumObjects, cli.NumVersions),
			IsConfirm: true,
		}
		result, err := prompt.Run()
		if err != nil || strings.ToLower(result) != "y" {
			fmt.Println("Command aborted!")
			return
		}
	}

	fmt.Println("")

	// Set up S3 client
	s3 := aws.NewS3Service(aws.WithAWSEndpoint(cli.AWSEndpoint))

	// Init progress bar
	uiprogress.Start()

	bucketsBar := uiprogress.AddBar(cli.NumBuckets).AppendCompleted().PrependElapsed()
	bucketsBar.PrependFunc(func(b *uiprogress.Bar) string {
		return "create buckets "
	})
	objectsBar := uiprogress.AddBar(cli.NumObjects).AppendCompleted().PrependElapsed()
	objectsBar.PrependFunc(func(b *uiprogress.Bar) string {
		return "create objects "
	})
	versionsBar := uiprogress.AddBar(cli.NumVersions).AppendCompleted().PrependElapsed()
	versionsBar.PrependFunc(func(b *uiprogress.Bar) string {
		return "create versions"
	})

	for ib := 0; ib < cli.NumBuckets; ib++ {
		// Create bucket
		bucketName := cli.BucketPrefix + "-" + uuid.NewString()
		var err error

		if cli.NumVersions > 1 {
			err = s3.CreateBucketSimple(context.TODO(), bucketName, cli.Region, true)
		} else {
			err = s3.CreateBucketSimple(context.TODO(), bucketName, cli.Region, false)
		}
		if err != nil {
			fmt.Printf("an error occured while creating a new bucket: %s\n", err.Error())
			fmt.Printf("bucket name: %s\n", bucketName)
			os.Exit(1)
		}
		bucketsBar.Incr()

		// Create Objects
		err = objectsBar.Set(0)
		if err != nil {
			fmt.Printf("UI error occured: %s", err.Error())
		}
		for io := 0; io < cli.NumObjects; io++ {
			r := strings.NewReader(generators.GeneratePhrase(20))
			key := uuid.NewString()
			_, _, err := s3.PutObjectSimple(context.TODO(), bucketName, key, r)
			if err != nil {
				fmt.Printf("an error occured while putting object: %s", err.Error())
				fmt.Printf("bucket name: %s, key name: %s\n", bucketName, key)
				os.Exit(1)
			}
			objectsBar.Incr()

			// Create Versions
			if cli.NumVersions > 1 {
				err = versionsBar.Set(1)
				if err != nil {
					fmt.Printf("UI error occured: %s", err.Error())
				}
				for iv := 1; iv < cli.NumObjects; iv++ {
					r = strings.NewReader(generators.GeneratePhrase(20))
					_, _, err := s3.PutObjectSimple(context.TODO(), bucketName, key, r)
					if err != nil {
						fmt.Printf("an error occured while putting new object version: %s", err.Error())
						fmt.Printf("bucket name: %s, key name: %s\n", bucketName, key)
						os.Exit(1)
					}
					versionsBar.Incr()
				}
			}
		}
	}

	uiprogress.Stop()

	fmt.Println("")
	fmt.Println("resources created!")
}
