package main

import (
	"github.com/alecthomas/kong"
)

var cli struct {
	Debug bool `help:"enable debug mode"`
}

func main() {

	ctx := kong.Parse(&cli,
		kong.Name("s3-nuke"),
		kong.Description("Quickly destroy all objects and versions in an AWS S3 bucket."))

	var err error
	ctx.FatalIfErrorf(err)

}
