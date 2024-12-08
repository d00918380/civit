package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/carlmjohnson/requests"
	"github.com/d00918380/civit/internal/civit"
)

var CLI struct {
	APIKey string `env:"CIVIT_API_KEY" help:"API key." required:""`
	Posts  struct {
		Download struct {
			Ids []int `arg:"" name:"id" help:"Post IDs to download."`
		} `cmd:"" help:"Download posts."`
	} `cmd:"" help:"Manage posts."`
	Users struct {
		Download struct {
			Username string `arg:"" name:"username" help:"Username to download."`
		} `cmd:"" help:"Download users."`
	} `cmd:"" help:"Manage users."`
	Images struct {
		Metadata struct {
			Username string `arg:"" name:"username" help:"Username to get metadata for."`
		} `cmd:"" help:"Get metadata for users."`
	} `cmd:"" help:"Manage images."`
	Report struct {
		Input string `arg:"" name:"input" help:"Input file."`
	} `cmd:"" help:"Generate a report."`
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
	case "images metadata <username>":
		items, err := civit.New(CLI.APIKey).ItemsForUser(context.Background(), CLI.Images.Metadata.Username)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(items)
	case "posts download <id>":
		client := civit.New(CLI.APIKey)
		for _, id := range CLI.Posts.Download.Ids {
			items, err := client.ItemsForPost(context.Background(), id)
			if err != nil {
				return err
			}
			return downloadItems(context.Background(), items)
		}
		return nil
	case "report <input>":
		var items []*civit.Item
		f, err := os.Open(CLI.Report.Input)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&items); err != nil {
			return err
		}
		return report(os.Stdout, items)
	case "users download <username>":
		items, err := civit.New(CLI.APIKey).ItemsForUser(context.Background(), CLI.Users.Download.Username)
		if err != nil {
			return err
		}
		return downloadItems(context.Background(), items)
	default:
		return fmt.Errorf("unknown command: %s", ctx.Command())
	}
}

func downloadItems(ctx context.Context, items []*civit.Item) error {
	for _, item := range items {
		path := imageToPath(item)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Downloading %s to %s\n", item.Url, path)
		if err := requests.URL(item.Url).ToFile(path).Fetch(ctx); err != nil {
			return err
		}
		ctime := item.CreatedAt
		if err := os.Chtimes(path, ctime, ctime); err != nil {
			return err
		}
		if err := os.Chtimes(filepath.Dir(path), ctime, ctime); err != nil {
			return err
		}
	}
	return nil
}

func imageToPath(image *civit.Item) string {
	return filepath.Join("posts", image.Username, fmt.Sprintf("%d", image.PostId), path.Base(image.Url))
}
