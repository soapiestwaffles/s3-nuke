package config

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		wantErr bool
	}{
		{
			name:    "endpoint not set",
			region:  "us-west-2",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("newConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Region != tt.region {
				t.Errorf("newConfig() got = %v, want %v", got.Region, "us-west-2")
			}
		})
	}
}
