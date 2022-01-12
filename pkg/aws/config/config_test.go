package config

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		awsEndpoint string
		wantErr     bool
	}{
		{
			name:        "endpoint not set",
			awsEndpoint: "",
			wantErr:     false,
		},
		{
			name:        "endpoint set",
			awsEndpoint: "http://test-endpoint.com:1234",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New("us-west-2", tt.awsEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("newConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.awsEndpoint == "" {
				if got.EndpointResolverWithOptions != nil {
					t.Errorf("%s: newConfig() error, EndpointResolverWithOptions was set but was supposed to be nil", tt.name)
				}
			} else {
				if got.EndpointResolverWithOptions == nil {
					t.Errorf("%s: newConfig() error, EndpointResolveWithOptions was nil but was supposed to be set", tt.name)
				}
			}
		})
	}
}
