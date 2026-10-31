// Command netbird-mcp is an MCP server for the NetBird management API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koinsaari/netbird-mcp/internal/config"
	"github.com/koinsaari/netbird-mcp/internal/netbird"
	"github.com/koinsaari/netbird-mcp/internal/tools"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// After the first signal, restore default handling so a second one kills the process.
	context.AfterFunc(ctx, stop)
	err := run(ctx, os.Args[1:], os.Getenv, &mcp.StdioTransport{}, os.Stdout, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "netbird-mcp:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string, transport mcp.Transport, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("netbird-mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse flags: %w", err)
	}
	if *showVersion {
		_, err := fmt.Fprintf(stdout, "netbird-mcp %s\n", version)
		return err
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	hc := &http.Client{
		Timeout: 30 * time.Second,
		// A redirect could forward the Authorization header elsewhere.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "netbird-mcp", Version: version}, &mcp.ServerOptions{Logger: logger})
	tools.Register(server, netbird.New(cfg.APIURL, cfg.Token, hc), logger)

	err = server.Run(ctx, transport)
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}
	return fmt.Errorf("run server: %w", err)
}
