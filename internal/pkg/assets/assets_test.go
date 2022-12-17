package assets

import (
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name      string
		assetFile string
		assetVar  *string
	}{
		{
			name:      "logo",
			assetFile: "logo.txt",
			assetVar:  &Logo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.assetFile)
			if err != nil {
				t.Errorf("Asset file: %s - error reading asset file: %v", tt.assetFile, err)
			}
			t.Logf("Asset file: %s - successfully read asset file", tt.assetFile)
			if *tt.assetVar != string(content) {
				t.Errorf("Asset file: file contents of %s does not match asset var", tt.assetFile)
			}
		})
	}
}
