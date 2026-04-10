package batch

import (
	"testing"
)

func TestNewClientFactory(t *testing.T) {
	tests := []struct {
		name      string
		stage     string
		wantType  string
		wantError bool
	}{
		{
			name:      "moto stage returns LocalStackClient",
			stage:     "moto",
			wantType:  "*LocalStackClient",
			wantError: false,
		},
		{
			name:      "localstack stage returns LocalStackClient",
			stage:     "localstack",
			wantType:  "*LocalStackClient",
			wantError: false,
		},
		{
			name:      "mirror stage returns AWSClient",
			stage:     "mirror",
			wantType:  "*AWSClient",
			wantError: false,
		},
		{
			name:      "invalid stage returns error",
			stage:     "invalid",
			wantType:  "",
			wantError: true,
		},
		{
			name:      "empty stage returns error",
			stage:     "",
			wantType:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.stage)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("NewClient(%q) expected error, got nil", tt.stage)
				}
				return
			}
			
			if err != nil {
				t.Errorf("NewClient(%q) unexpected error: %v", tt.stage, err)
				return
			}
			
			// Check the type of the returned client
			clientType := ""
			switch client.(type) {
			case *LocalStackClient:
				clientType = "*LocalStackClient"
			case *AWSClient:
				clientType = "*AWSClient"
			default:
				t.Errorf("NewClient(%q) returned unexpected type: %T", tt.stage, client)
				return
			}
			
			if clientType != tt.wantType {
				t.Errorf("NewClient(%q) = %v, want %v", tt.stage, clientType, tt.wantType)
			}
		})
	}
}