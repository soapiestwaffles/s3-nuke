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

func TestNewWithProfile(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		profile string
		wantErr bool
	}{
		{
			name:    "no profile specified",
			region:  "us-west-2",
			profile: "",
			wantErr: false,
		},
		{
			name:    "profile specified (may fail if profile doesn't exist)",
			region:  "us-east-1",
			profile: "default",
			wantErr: true, // Expected to fail in test environment without AWS config
		},
		{
			name:    "invalid profile name",
			region:  "us-east-1",
			profile: "non-existent-profile",
			wantErr: true, // Expected to fail with non-existent profile
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewWithProfile(tt.region, tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWithProfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Only check region if no error occurred
			if err == nil && got.Region != tt.region {
				t.Errorf("NewWithProfile() got region = %v, want %v", got.Region, tt.region)
			}
		})
	}
}
