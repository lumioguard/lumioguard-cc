// Command lumioguard-cc is the lumioguard CC (Crap Cleaner) entry point.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/lumioguard/lumioguard-cc/internal/cli"
	"github.com/lumioguard/lumioguard-cc/internal/compose"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.New(compose.NewApplication()).Execute(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	// Released explicitly rather than deferred, because os.Exit skips defers.
	stop()
	os.Exit(code)
}
