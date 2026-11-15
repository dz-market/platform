package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dz-market/platform/internal/repotools/internal/release"
)

func main() {
	root := flag.String("root", "../..", "repository root")
	ref := flag.String("ref", os.Getenv("RELEASE_REF"), "commit, branch or tag to publish")

	flag.Parse()

	if flag.NArg() != 0 || *ref == "" {
		_, _ = fmt.Fprintln(os.Stderr, "usage: tagrelease -root <path> -ref <ref>")

		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := release.Run(ctx, *root, *ref)

	stop()

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
