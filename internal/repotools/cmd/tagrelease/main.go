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
	releaseTag := flag.String("release-tag", os.Getenv("RELEASE_TAG"), "existing release tag")

	flag.Parse()

	if flag.NArg() != 0 || *releaseTag == "" {
		_, _ = fmt.Fprintln(os.Stderr, "usage: tagrelease -root <path> -release-tag <tag>")

		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := release.Run(ctx, *root, *releaseTag)

	stop()

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
