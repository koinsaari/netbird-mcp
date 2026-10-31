package main

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantStdout string
		wantStderr string
	}{
		{name: "version flag", args: []string{"-version"}, wantStdout: "netbird-mcp " + version + "\n"},
		{name: "no args", args: nil, wantStderr: "no tools yet"},
		{name: "unknown flag", args: []string{"-bogus"}, wantErr: true, wantStderr: "bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr strings.Builder
			err := run(tt.args, &stdout, &stderr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}
