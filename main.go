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
		url := "https://civitai.com/api/v1/images?nsfw=X&token=" + CLI.APIKey + "&username=" + CLI.Images.Metadata.Username
		images, err := fetchItems(context.Background(), url)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(images)
	case "posts download <id>":
		for _, id := range CLI.Posts.Download.Ids {
			url := "https://civitai.com/api/v1/images?nsfw=X&token=" + CLI.APIKey + "&postId=" + fmt.Sprintf("%d", id)
			if err := downloadImages(context.Background(), url); err != nil {
				return err
			}
		}
		return nil
	case "users download <username>":
		url := "https://civitai.com/api/v1/images?nsfw=X&token=" + CLI.APIKey + "&username=" + CLI.Users.Download.Username
		return downloadImages(context.Background(), url)
	default:
		return fmt.Errorf("unknown command: %s", ctx.Command())
	}
}

func downloadImages(ctx context.Context, url string) error {
	var resp struct {
		Items    []*civit.Item
		Metadata civit.Metadata `json:"metadata"`
	}
	err := requests.URL(url).ToJSON(&resp).Fetch(ctx)
	if err != nil {
		return err
	}
	for _, item := range resp.Items {
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
	if resp.Metadata.NextPage != "" {
		return downloadImages(ctx, resp.Metadata.NextPage)
	}
	return nil
}

func imageToPath(image *civit.Item) string {
	return filepath.Join("posts", image.Username, fmt.Sprintf("%d", image.PostId), path.Base(image.Url))
}

func fetchItems(ctx context.Context, url string) ([]*civit.Item, error) {
	var resp struct {
		Items    []*civit.Item
		Metadata civit.Metadata `json:"metadata"`
	}
	err := requests.URL(url).ToJSON(&resp).Fetch(ctx)
	if err != nil {
		return nil, err
	}
	images := resp.Items
	if resp.Metadata.NextPage != "" {
		nextImages, err := fetchItems(ctx, resp.Metadata.NextPage)
		if err != nil {
			return nil, err
		}
		images = append(images, nextImages...)
	}
	return images, nil
}
