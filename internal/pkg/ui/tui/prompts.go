package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/soapiestwaffles/s3-nuke/internal/pkg/generators"
	"github.com/soapiestwaffles/s3-nuke/pkg/aws/s3"
)

// SelectBucketsPrompt will create the UI select element for the user to select a bucket from a list
func SelectBucketsPrompt(buckets []s3.Bucket) (string, error) {
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
		name := strings.ReplaceAll(strings.ToLower(bucket.Name), " ", "")
		input = strings.ReplaceAll(strings.ToLower(input), " ", "")

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     "Select a bucket to nuke:",
		Items:     derefBucket,
		Templates: templates,
		Size:      5,
		Searcher:  searcher,
		Stdout:    &bellSkipper{},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return "", nil
	}

	return *buckets[i].Name, nil
}

// TypeMatchingPhrase presents the user with a randomized "fakelish" phrase which they have to retype to continue
func TypeMatchingPhrase() bool {
	phrase := generators.GeneratePhrase(4)

	fmt.Println("Please enter the following phrase to continue:", phrase)
	prompt := promptui.Prompt{
		Label: "Enter phrase",
	}

	result, err := prompt.Run()
	if err != nil {
		return false
	}

	if strings.ToLower(result) != phrase {
		return false
	}

	return true
}

// ---

// bellSkipper implements an io.WriteCloser that skips the terminal bell
// character (ASCII code 7), and writes the rest to os.Stderr. It is used to
// replace readline.Stdout, that is the package used by promptui to display the
// prompts.
//
// This is a workaround for the bell issue documented in
// https://github.com/manifoldco/promptui/issues/49.
type bellSkipper struct{}

// Write implements an io.WriterCloser over os.Stderr, but it skips the terminal
// bell character.
func (bs *bellSkipper) Write(b []byte) (int, error) {
	const charBell = 7 // c.f. readline.CharBell
	if len(b) == 1 && b[0] == charBell {
		return 0, nil
	}
	return os.Stderr.Write(b)
}

// Close implements an io.WriterCloser over os.Stderr.
func (bs *bellSkipper) Close() error {
	return os.Stderr.Close()
}
