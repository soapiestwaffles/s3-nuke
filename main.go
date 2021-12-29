package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/assets"
)

var cli struct {
	Debug bool `help:"enable debug mode"`
}

func main() {

	ctx := kong.Parse(&cli,
		kong.Name("s3-nuke"),
		kong.Description("Quickly destroy all objects and versions in an AWS S3 bucket."))

	fmt.Println(assets.Logo)

	var err error
	ctx.FatalIfErrorf(err)
}
