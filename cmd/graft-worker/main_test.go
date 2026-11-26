package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		canceled   bool
		args       []string
		wantSubstr string
		wantErr    bool
	}{
		{
			name:       "prints version summary",
			args:       []string{"-version"},
			wantSubstr: "graft-worker version=",
		},
		{
			name:       "runs default bootstrap startup",
			args:       nil,
			wantSubstr: "graft-worker bootstrap ready",
		},
		{
			name:    "rejects unknown flag",
			args:    []string{"-unknown-flag"},
			wantErr: true,
		},
		{
			name:     "honors canceled context",
			canceled: true,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.canceled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			var out bytes.Buffer
			err := run(ctx, tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("run(%v) expected error, got nil", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("run(%v) unexpected error: %v", tc.args, err)
			}
			if !strings.Contains(out.String(), tc.wantSubstr) {
				t.Fatalf("run(%v) output = %q, want substring %q", tc.args, out.String(), tc.wantSubstr)
			}
		})
	}
}
