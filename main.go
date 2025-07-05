package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/carlmjohnson/requests"
	"github.com/d00918380/civit/internal/algorithms"
	"github.com/d00918380/civit/internal/trpc"
)

var CLI struct {
	APIKey  string `env:"CIVIT_API_KEY" help:"API key." required:""`
	Cookies string `help:"Path to the cookies file." default:"cookies.json"`
	Posts   struct {
		Download struct {
			Ids []int `arg:"" name:"id" help:"Post IDs to download."`
		} `cmd:"" help:"Download posts."`
	} `cmd:"" help:"Manage posts."`
	Users struct {
		Following struct {
		} `cmd:"" help:"Fetch users that the current user is following."`
		Download struct {
			Username string `arg:"" name:"username" help:"Username to download."`
			Id       int    `arg:"" name:"id" help:"User ID to download."`
		} `cmd:"" help:"Download users."`
	} `cmd:"" help:"Manage users."`
	User struct {
		List struct {
			Username string `arg:"" name:"username" help:"Username to list followers for."`
		} `cmd:"" help:"List management."`
	} `cmd:"" help:"Manage user."`
	Images struct {
		Metadata struct {
			Username string `arg:"" name:"username" help:"Username to get metadata for."`
			Id       int    `arg:"" name:"id" help:"User ID to get metadata for."`
		} `cmd:"" help:"Get metadata for users."`
	} `cmd:"" help:"Manage images."`
	Orchestrator struct {
		Download struct {
		} `cmd:"" help:"Download all images in the orchestrator."`
	} `cmd:"" help:"Manage orchestrator."`
	Report struct {
		Input string `arg:"" name:"input" help:"Input file."`
	} `cmd:"" help:"Generate a report."`
	CSV struct {
		Input string `arg:"" name:"input" help:"Input file."`
	} `cmd:"" help:"Generate a CSV."`
	Reactions struct {
		Images string        `help:"path to the file with images." default:"images.txt"`
		Models string        `help:"path to the file with models." default:"models.txt"`
		Whales string        `help:"path to the file with whales." default:"whales.txt"`
		Delay  time.Duration `help:"delay between runs." default:"20m"`
	} `cmd:"" help:"Manage reactions."`
	Showcase struct {
		Add struct {
			Images []int `arg:"" name:"images" help:"Image IDs to add to the showcase."`
		} `cmd:"" help:"Add images to the showcase."`
		Leaderboard struct {
			Input string `arg:"" name:"input" help:"Input file."`
		} `cmd:"" help:"Set showcase the leaderboard."`
	} `cmd:"" help:"Manage showcase."`
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
	case "images metadata <username> <id>":
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		ctx := context.Background()
		var items []trpc.Item
		iter := c.ImagesForUser(ctx, CLI.Images.Metadata.Username, CLI.Images.Metadata.Id)
		for iter.Next() {
			items = append(items, iter.Item())
		}
		if err := iter.Err(); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(items)

	case "orchestrator download":
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		ctx := context.Background()
		iter := c.QueryGeneratedImages(ctx)
		for iter.Next() {
			item := iter.Item()
			for _, step := range item.Steps {
				for _, image := range step.Images {
					ext := filepath.Ext(image.ID)
					if ext == "" {
						ext = ".jpeg"
					} else {
						ext = ""
					}
					path := filepath.Join(
						"generated",
						fmt.Sprintf("%04d", image.Completed.Year()),
						fmt.Sprintf("%02d", image.Completed.Month()),
						fmt.Sprintf("%02d", image.Completed.Day()),
						image.ID+ext,
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

	case "users download <username> <id>":
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		ctx := context.Background()
		iter := c.ImagesForUser(ctx, CLI.Images.Metadata.Username, CLI.Images.Metadata.Id)
		for iter.Next() {
			img := iter.Item()
			path := filepath.Join(
				"posts",
				strconv.Itoa(img.PostID),
				img.URL+".jpeg",
			)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			name := strings.TrimPrefix(img.URL, "/") // some images have a leading slash??
			url := fmt.Sprintf("https://image.civitai.com/xG1nkqKTMzGDvpLrqFT7WA/%s/%s.jpeg", img.URL, name)
			fmt.Println("Downloading", url, "to", path)
			if err := requests.URL(url).ToFile(path).Fetch(ctx); err != nil {
				fmt.Println(err) // some images are missing
				fmt.Printf("%+v\n", img)
			}
		}
		return iter.Err()
	case "posts download <id>":
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		ctx := context.Background()
		for _, id := range CLI.Posts.Download.Ids {
			iter := c.ImagesForPost(ctx, id)
			for iter.Next() {
				img := iter.Item()
				path := filepath.Join(
					"posts",
					strconv.Itoa(img.PostID),
					img.URL+".jpeg",
				)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					return err
				}
				name := strings.TrimPrefix(img.URL, "/") // some images have a leading slash??
				url := fmt.Sprintf("https://image.civitai.com/xG1nkqKTMzGDvpLrqFT7WA/%s/%s.jpeg", img.URL, name)
				fmt.Println("Downloading", url, "to", path)
				if err := requests.URL(url).ToFile(path).Fetch(ctx); err != nil {
					fmt.Println(err) // some images are missing
					fmt.Printf("%+v\n", img)
				}
			}
		}
		return nil
	case "report <input>":
		var items []*trpc.Item
		f, err := os.Open(CLI.Report.Input)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&items); err != nil {
			return err
		}
		items = algorithms.Filter(items, func(img *trpc.Item) bool {
			return img.Published()
		})
		return report(os.Stdout, items)
	case "csv <input>":
		var items []*trpc.Item
		f, err := os.Open(CLI.CSV.Input)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&items); err != nil {
			return err
		}
		items = algorithms.Filter(items, func(img *trpc.Item) bool {
			return img.Published()
		})
		return csv(os.Stdout, items)
	case "reactions":
		reactions := &ReactionsProcessor{
			imagesFile: CLI.Reactions.Images,
			modelsFile: CLI.Reactions.Models,
			whalesFile: CLI.Reactions.Whales,
			trpc:       trpc.New(CLI.APIKey, CLI.Cookies),
		}
		for {
			err := reactions.Run()
			if err != nil {
				return err
			}
			log.Printf("sleeping til %v", time.Now().Add(CLI.Reactions.Delay))
			time.Sleep(CLI.Reactions.Delay)
		}
	case "showcase add <images>":
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		ctx := context.Background()
		for _, id := range CLI.Showcase.Add.Images {
			if err := c.AddImageToShowcase(ctx, id); err != nil {
				return err
			}
			fmt.Printf("Added %d to showcase\n", id)
		}
		return nil
	case "showcase leaderboard <input>":
		var items []*trpc.Item
		f, err := os.Open(CLI.Showcase.Leaderboard.Input)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&items); err != nil {
			return err
		}
		items = algorithms.Filter(items, func(img *trpc.Item) bool {
			return img.Published()
		})
		const IMAGE_SCORE_FALLOFF = 120

		cutoff := time.Now().Add(-time.Hour * 24 * 30) // 30 days ago
		var entries []*LeaderboardEntry
		for _, i := range (&data{Items: items}).Images() {
			if i.PublishedAt.After(cutoff) {
				entries = append(entries, &LeaderboardEntry{
					image: i,
				})
			}
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Score() > entries[j].Score()
		})
		for rank, e := range entries {
			quantityMultiplier := math.Max(0, 1-math.Pow(float64(rank)/IMAGE_SCORE_FALLOFF, 0.5))
			score := float64(e.Score())
			// fmt.Println("rank", rank, "score", score, "quantityMultiplier", quantityMultiplier, "adjustedScore", score*quantityMultiplier)
			e.AdjustedScore = score * quantityMultiplier
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].AdjustedScore > entries[j].AdjustedScore
		})
		// the scoreboard ignores duplicates, but clearing it involves reposting the profile struct,
		// instead, push at least 60 images to the showcase which should mean that by the 30'th image
		// the top 30 images are _not_ in the showcase and thus will flow in as expected.
		entries = entries[:min(60, len(entries))]
		slices.Reverse(entries)
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		for _, e := range entries {
			if err := c.AddImageToShowcase(context.Background(), e.image.ID); err != nil {
				return err
			}
			fmt.Printf("Added %d to showcase\n", e.image.ID)
		}
		return nil

	case "user list <username>":
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		ctx := context.Background()
		lists, err := c.ListsForUser(ctx, CLI.User.List.Username)
		if err != nil {
			return err
		}
		followers := algorithms.Map(lists.Followers, func(u trpc.User) string { return u.Username })
		following := algorithms.Map(lists.Following, func(u trpc.User) string { return u.Username })
		fmt.Println("Followers:", len(followers), followers)
		fmt.Println("Following:", len(following), following)

		m := map[string]bool{}
		for _, u := range lists.Followers {
			m[u.Username] = true
		}
		for _, u := range lists.Following {
			if !m[u.Username] {
				fmt.Println("Not mutual:", u.Username)
			}
		}
		return nil
	// case "user list following <username>":
	// 	c := trpc.New(CLI.APIKey)
	// 	ctx := context.Background()
	// 	iter := c.ListForUser(ctx, CLI.User.List.Following.Username, "following")
	// 	for iter.Next() {
	// 		list := iter.Item()
	// 		fmt.Println(list)
	// 	}
	// 	return iter.Err()
	case "users following":
		c := trpc.New(CLI.APIKey, CLI.Cookies)
		ctx := context.Background()
		iter := c.UsersFollowing(ctx)
		for iter.Next() {
			user := iter.Item()
			fmt.Println(user)
		}
		return iter.Err()
	default:
		return fmt.Errorf("unknown command: %s", ctx.Command())
	}
}
