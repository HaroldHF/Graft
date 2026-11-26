package buildinfo_test

import (
	"strings"
	"testing"

	"github.com/HaroldHF/Graft/internal/buildinfo"
)

func TestSummary(t *testing.T) {
	tests := []struct {
		name      string
		component string
		wantParts []string
	}{
		{
			name:      "control plane binary",
			component: "graft-api",
			wantParts: []string{"graft-api", "version=", "commit=", "built="},
		},
		{
			name:      "worker binary",
			component: "graft-worker",
			wantParts: []string{"graft-worker", "version=", "commit=", "built="},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildinfo.Summary(tc.component)
			for _, part := range tc.wantParts {
				if !strings.Contains(got, part) {
					t.Fatalf("Summary(%q) = %q, missing %q", tc.component, got, part)
				}
			}
		})
	}
}
