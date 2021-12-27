package assets

import "embed"

//go:embed logo.txt
var f embed.FS

var (
	// Logo contains the ascii s3-nuke logo
	Logo string
)

func init() {
	// Logo
	rawLogo, _ := f.ReadFile("logo.txt")
	Logo = string(rawLogo)
}
