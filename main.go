package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/d00918380/civit/internal/civit"
)

var CLI struct {
	APIKey string `env:"CIVIT_API_KEY" help:"API key."`
	Posts  struct {
		Download struct {
			Ids []int `arg:"" name:"id" help:"Post IDs to download."`
		} `cmd:"" help:"Download posts."`
	} `cmd:"" help:"Manage posts."`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "posts download <id>":
		return civit.NewPosts(civit.NewClient(CLI.APIKey)).Download(CLI.Posts.Download.Ids...)
	default:
		return fmt.Errorf("unknown command: %s", ctx.Command())
	}
}
