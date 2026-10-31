package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func withToken(k string) string {
	if k == "NETBIRD_TOKEN" {
		return "secret-token"
	}
	return ""
}

// eofTransport is a transport whose input is already closed, like stdin at EOF.
func eofTransport() mcp.Transport {
	inR, inW := io.Pipe()
	_ = inW.Close() // io.Pipe Close never fails
	_, outW := io.Pipe()
	return &mcp.IOTransport{Reader: inR, Writer: outW}
}

func TestRun(t *testing.T) {
	noEnv := func(string) string { return "" }
	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	tests := []struct {
		name       string
		ctx        context.Context
		args       []string
		getenv     func(string) string
		wantErr    string
		wantStdout string
		wantStderr string
	}{
		{name: "version without token", ctx: t.Context(), args: []string{"-version"}, getenv: noEnv, wantStdout: "netbird-mcp " + version + "\n"},
		{name: "unknown flag", ctx: t.Context(), args: []string{"-bogus"}, getenv: withToken, wantErr: "bogus", wantStderr: "bogus"},
		{name: "missing token", ctx: t.Context(), getenv: noEnv, wantErr: "NETBIRD_TOKEN"},
		{name: "stdin EOF", ctx: t.Context(), getenv: withToken},
		{name: "canceled context", ctx: canceled, getenv: withToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr strings.Builder
			err := run(tt.ctx, tt.args, tt.getenv, eofTransport(), &stdout, &stderr)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want to contain %q", stderr.String(), tt.wantStderr)
			}
			if strings.Contains(stderr.String(), "secret-token") {
				t.Error("stderr leaked token")
			}
		})
	}
}

func TestRunListsTools(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, nil, withToken, &mcp.IOTransport{Reader: stdinR, Writer: stdoutW}, io.Discard, io.Discard)
		_ = stdoutW.Close() // io.Pipe Close never fails
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: stdoutR, Writer: stdinW}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	found := false
	for _, tool := range res.Tools {
		found = found || tool.Name == "list_peers"
	}
	if !found {
		t.Errorf("list_peers not in %v", res.Tools)
	}
	if err := session.Close(); err != nil {
		t.Logf("close: %v", err)
	}
	if err := <-done; err != nil {
		t.Errorf("run: %v", err)
	}
}
