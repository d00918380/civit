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
	"github.com/d00918380/civit/internal/trpc"
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
	Orchestrator struct {
		Download struct {
		} `cmd:"" help:"Download all images in the orchestrator."`
	} `cmd:"" help:"Manage orchestrator."`
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
	case "orchestrator download":
		c := trpc.New(CLI.APIKey)
		ctx := context.Background()
		iter := c.QueryGeneratedImages(ctx)
		for iter.Next() {
			item := iter.Item()
			for _, step := range item.Steps {
				for _, image := range step.Images {
					path := filepath.Join(
						"generated",
						fmt.Sprintf("%04d", image.Completed.Year()),
						fmt.Sprintf("%02d", image.Completed.Month()),
						fmt.Sprintf("%02d", image.Completed.Day()),
						item.ID,
						image.ID+".jpg",
					)
					fmt.Println(image.URL, path)
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						return err
					}
					if err := requests.URL(image.URL).ToFile(path).Fetch(ctx); err != nil {
						fmt.Println(err) // some images are missing
					}
				}
			}
		}
		return iter.Err()
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
