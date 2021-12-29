package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/assets"
)

const releaseURL = "https://github.com/soapiestwaffles/s3-nuke/release"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	// builtBy = "unknown"

	cli struct {
		Debug   bool `help:"enable debug mode"`
		Version bool `help:"display version information"`
	}
)

func main() {

	ctx := kong.Parse(&cli,
		kong.Name("s3-nuke"),
		kong.Description("Quickly destroy all objects and versions in an AWS S3 bucket."))

	fmt.Println(assets.Logo)
	if cli.Version {
		fmt.Println("Find releases at", releaseURL)
		fmt.Println("")
		fmt.Println("version....:", version)
		fmt.Println("commit.....:", commit)
		fmt.Println("data.......:", date)
		os.Exit(0)
	}

	var err error
	ctx.FatalIfErrorf(err)
}
