package buildfeed

import "testing"

func TestPreparePublishingPolicy(t *testing.T) {
	tests := []struct {
		name      string
		input     Event
		wantEvent string
		wantErr   bool
	}{
		{
			name:      "failed build is visible to room subscribers",
			input:     Event{Kind: "build", Repository: "acme/compiler", Ref: "main", Status: "failed", Summary: "linux test failed"},
			wantEvent: "build.failed",
		},
		{
			name:      "published release is visible to room subscribers",
			input:     Event{Kind: "release", Repository: "acme/compiler", Ref: "v1.8.0", Status: "published"},
			wantEvent: "release.published",
		},
		{
			name:    "unknown transition is rejected before publishing",
			input:   Event{Kind: "release", Repository: "acme/compiler", Ref: "v1.8.0", Status: "failed"},
			wantErr: true,
		},
		{
			name:    "repository is required",
			input:   Event{Kind: "diagnostic", Status: "warning"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Prepare(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Event != tt.wantEvent {
				t.Fatalf("Prepare() event = %q, want %q", got.Event, tt.wantEvent)
			}
		})
	}
}
