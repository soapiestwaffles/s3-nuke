package main

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/manifoldco/promptui"
)

var (
	cli struct {
		AWSEndpoint string `help:"override AWS endpoint address" short:"e" optional:"" env:"AWS_ENDPOINT"`
		NumBuckets  int    `help:"number of buckets with randomized names to create" short:"n" required:""`
		NumObjects  int    `help:"number of random objects generated and put into buckets" short:"o" required:""`
		NumVersions int    `help:"number of versions to create for each random object" short:"v" required:""`
		YesFlag     bool   `name:"yes" help:"bypass user prompt and proceed with action automatically" optional:""`
	}
)

func main() {
	ctx := kong.Parse(&cli,
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

	// Set up S3 client
	// s3 := aws.NewS3Service(aws.WithAWSEndpoint(cli.AWSEndpoint))

	var err error
	ctx.FatalIfErrorf(err)
}
