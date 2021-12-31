package main

import (
	"testing"
)

func Test_newAWSSDKConfig(t *testing.T) {
	tests := []struct {
		name        string
		awsEndpoint string
		wantNil     bool
		wantErr     bool
	}{
		{
			name:        "endpoint not set",
			awsEndpoint: "",
			wantNil:     true,
			wantErr:     false,
		},
		{
			name:        "endpoint set",
			awsEndpoint: "http://test-endpoint.com:1234",
			wantNil:     true,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newAWSSDKConfig(tt.awsEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("newAWSSDKConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil {
				if got.EndpointResolverWithOptions != nil {
					t.Errorf("newAWSSDKConfig() error, EndpointResolverWithOptions was set")
				}
			} else {
				if got.EndpointResolverWithOptions == nil {
					t.Errorf("newAWSSDKConfig() error, EndpointResolveWithOptions was not set")
				}
			}
		})
	}
}
