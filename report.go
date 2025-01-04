package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/d00918380/civit/internal/algorithms"
	"github.com/d00918380/civit/internal/civit"
	"github.com/montanaflynn/stats"
)

//go:embed report.html
var reportHTML string

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func report(w io.Writer, items []*civit.Item) error {
	data := &data{
		Items: items,
	}

	funcs := template.FuncMap{
		"epoch": func() time.Time {
			// two years ago
			return time.Now().Add(-time.Hour * 24 * 365 * 2)
		},
		"mean":       stats.Mean,
		"sum":        stats.Sum,
		"percentile": stats.Percentile,
		"stddev":     stats.StandardDeviation,
		"json": func(v any) (template.JS, error) {
			var buf bytes.Buffer
			err := json.NewEncoder(&buf).Encode(v)
			s := strings.TrimSpace(buf.String()) // stupid json.NewEncoder adds a newline
			return template.JS(s), err
		},
		"ago": func(s string) (time.Time, error) {
			d, err := time.ParseDuration(s)
			if err != nil {
				return time.Time{}, err
			}
			return time.Now().Add(-d), nil
		},
		"between_range": func(a, b time.Time) TimeRange {
			return TimeRange{a, b}
		},
		"best_posts_per_day": func(n int) map[time.Time][]*post {
			days := make(map[time.Time][]*post)
			for _, p := range data.Posts() {
				day := p.PublishedAt().Truncate(time.Hour * 24)
				days[day] = append(days[day], p)
			}
			for day := range days {
				slices.SortStableFunc(days[day], func(a, b *post) int {
					return b.Score() - a.Score()
				})
				days[day] = days[day][:min(n, len(days[day]))]
			}
			return days
		},
		"worst_posts_per_day": func(n int) map[time.Time][]*post {
			days := make(map[time.Time][]*post)
			for _, p := range data.Posts() {
				day := p.PublishedAt().Truncate(time.Hour * 24)
				days[day] = append(days[day], p)
			}
			for day := range days {
				slices.SortStableFunc(days[day], func(a, b *post) int {
					return a.Score() - b.Score()
				})
				days[day] = days[day][:min(n, len(days[day]))]
			}
			return days
		},

		"per_day": func(images []*image) map[time.Time][]*image {
			days := make(map[time.Time][]*image)
			for _, i := range images {
				day := i.CreatedAt.Truncate(time.Hour * 24)
				days[day] = append(days[day], i)
			}
			return days
		},
		"worst_efficiency": func(r TimeRange, n int) []*post {
			insideRange := func(t time.Time) bool {
				return t.After(r.Start) && t.Before(r.End)
			}

			posts := data.PostsByEfficiency()
			posts = algorithms.Filter(posts, func(p *post) bool {
				return insideRange(p.PublishedAt())
			})
			slices.SortStableFunc(posts, func(a, b *post) int {
				return int((a.Efficiency() - b.Efficiency()) * 100)
			})
			return posts[:n]
		},
		"by_score": func(images []*image) []*image {
			slices.SortStableFunc(images, func(a, b *image) int {
				return b.Score() - a.Score()
			})
			return images
		},
		"by_date": func(images []*image) []*image {
			slices.SortStableFunc(images, func(a, b *image) int {
				return int(b.CreatedAt.Sub(a.CreatedAt))
			})
			return images
		},
		"between": func(a, b time.Time, images []*image) []*image {
			var result []*image
			for _, i := range images {
				if i.CreatedAt.After(a) && i.CreatedAt.Before(b) {
					result = append(result, i)
				}
			}
			return result
		},
		"reverse": func(images []*image) []*image {
			slices.Reverse(images)
			return images
		},
		"take": func(n int, images []*image) []*image {
			return images[:min(n, len(images))]
		},
		// less_thank returns images with a score less than n
		"less_than": func(n int, images []*image) []*image {
			return Filter(images, func(i *image) bool {
				return i.Score() < n
			})
		},
		"scores": func(images []*image) stats.Float64Data {
			return Map(images, func(i *image) float64 {
				return float64(i.Score())
			})
		},
		"by_post": func(images []*image) []*post {
			posts := make(map[int][]*image)
			for _, i := range images {
				posts[i.PostId] = append(posts[i.PostId], i)
			}
			var ps []*post
			for _, images := range posts {
				ps = append(ps, &post{images})
			}
			return ps
		},
		"by_post_date": func(posts []*post) []*post {
			slices.SortStableFunc(posts, func(a, b *post) int {
				return int(b.PublishedAt().Sub(a.PublishedAt()))
			})
			return posts
		},
	}

	t, err := template.New("report.html").Funcs(funcs).Parse(reportHTML)
	if err != nil {
		return err
	}

	return t.Execute(w, data)
}

type data struct {
	Items []*civit.Item
}

func (d *data) Images() []*image {
	return Map(d.Items, func(i *civit.Item) *image {
		return &image{i}
	})
}

func (d *data) Posts() []*post {
	posts := make(map[int]*post)
	for _, i := range d.Items {
		if _, ok := posts[i.PostId]; !ok {
			posts[i.PostId] = &post{}
		}
		posts[i.PostId].images = append(posts[i.PostId].images, &image{i})
	}
	var ps []*post
	for _, p := range posts {
		slices.SortStableFunc(p.images, func(a, b *image) int {
			return a.Index - b.Index
		})
		ps = append(ps, p)
	}
	return ps
}

func (d *data) PostsByEfficiency() []*post {
	posts := d.Posts()
	slices.SortStableFunc(posts, func(a, b *post) int {
		return int((b.Efficiency() - a.Efficiency()) * 100)
	})
	return posts
}

func (d *data) PostsByScore() []*post {
	posts := d.Posts()
	slices.SortStableFunc(posts, func(a, b *post) int {
		return b.Score() - a.Score()
	})
	return posts
}

func (d *data) PostsByDate() []*post {
	posts := d.Posts()
	slices.SortStableFunc(posts, func(a, b *post) int {
		return int(b.PublishedAt().Sub(a.PublishedAt()))
	})
	return posts
}

func (d *data) ImagesJSON() any {
	return Map(d.Images(), func(i *image) any {
		return map[string]any{
			"id":        i.Id,
			"imageURL":  i.ImageURL(),
			"postURL":   i.PostURL(),
			"score":     i.Score(),
			"createdAt": i.CreatedAt,
		}
	})
}

func (d *data) User() string {
	return First(d.Items).Username
}

// PostsJS prepares a fragment of javascript which represents the array of posts.
// Becauses its javascript, the final element in the array must not have a trailing comma. cool.
func (d *data) PostsJS() template.JS {
	var buf bytes.Buffer
	buf.WriteString("[\n")
	for i, p := range d.Posts() {
		buf.WriteString(fmt.Sprintf(`{id: %d, postURL: %q, score: %d, createdAt: new Date(%q)}`,
			p.Id(), p.PostURL(), p.Score(), First(p.images).CreatedAt.Format(time.RFC3339)))
		if i < len(d.Posts())-1 {
			buf.WriteString(",\n")
		}
	}
	buf.WriteString("\n];")
	return template.JS(buf.String())
}

// ImagesJS prepares a fragment of javascript which represents the array of images.
// Becauses its javascript, the final element in the array must not have a trailing comma. cool.
func (d *data) ImagesJS() template.JS {
	var buf bytes.Buffer
	buf.WriteString("[\n")
	for i, img := range d.Images() {
		buf.WriteString(fmt.Sprintf(`{id: %d, imageURL: %q, postURL: %q, score: %d, createdAt: new Date(%q)}`,
			img.Id, img.ImageURL(), img.PostURL(), img.Score(), img.CreatedAt))
		if i < len(d.Images())-1 {
			buf.WriteString(",\n")
		}
	}
	buf.WriteString("\n];")
	return template.JS(buf.String())
}

func (d *data) PostsJSON() any {
	return Map(d.Posts(), func(p *post) any {
		return map[string]any{
			"id":        p.Id(),
			"postURL":   p.PostURL(),
			"score":     p.Score(),
			"createdAt": First(p.images).CreatedAt,
		}
	})
}

func (d *data) Leaderboard() *Leaderboard {
	const IMAGE_SCORE_FALLOFF = 120

	cutoff := time.Now().Add(-time.Hour * 24 * 30) // 30 days ago
	var entries []*LeaderboardEntry
	for _, i := range d.Images() {
		if i.CreatedAt.After(cutoff) {
			entries = append(entries, &LeaderboardEntry{
				image: i,
			})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].AdjustedScore > entries[j].AdjustedScore
	})
	for rank, e := range entries {
		quantityMultiplier := math.Max(0, 1-math.Pow(float64(rank)/IMAGE_SCORE_FALLOFF, 0.5))
		score := float64(e.Score())
		// fmt.Println("rank", rank, "score", score, "quantityMultiplier", quantityMultiplier, "adjustedScore", score*quantityMultiplier)
		e.AdjustedScore = score * quantityMultiplier
	}

	// cutoff at 120
	entries = entries[:min(120, len(entries))]
	return &Leaderboard{Entries: entries}
}

type image struct {
	*civit.Item
}

func (i *image) Score() int {
	return Sum(i.Stats.CryCount, i.Stats.LaughCount, i.Stats.LikeCount, i.Stats.HeartCount, -i.Stats.DislikeCount)
}

func (i *image) ImageURL() template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf("https://civitai.com/images/%d", i.Id))
}

func (i *image) PostURL() template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf("https://civitai.com/posts/%d", i.PostId))
}

type post struct {
	images []*image
}

func (p *post) Id() int {
	return First(p.images).PostId
}

func (p *post) PostURL() template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf("https://civitai.com/posts/%d", First(p.images).PostId))
}

func (p *post) Score() int {
	return Sum(Map(p.images, func(i *image) int {
		return i.Score()
	})...)
}

func (p *post) PublishedAt() time.Time {
	// TODO: PubishedAt doesn't seem to parse correctly
	return First(p.images).CreatedAt
}

func (p *post) Images() []*image {
	return p.images
}

func (p *post) Efficiency() float64 {
	return float64(p.Score()) / float64(len(p.Images()))
}

type Leaderboard struct {
	Entries []*LeaderboardEntry
}

func (l *Leaderboard) Score() float64 {
	const IMAGE_SCORE_MULTIPLIER = 100
	sum := Sum(Map(l.Entries, func(e *LeaderboardEntry) float64 {
		return e.AdjustedScore
	})...)
	return math.Sqrt(sum) * IMAGE_SCORE_MULTIPLIER
}

type LeaderboardEntry struct {
	*image
	AdjustedScore float64
}

// Map applies the function f to each element of the slice and returns a new slice containing the results.
func Map[T, R any](s []T, f func(T) R) []R {
	r := make([]R, 0, len(s))
	for _, v := range s {
		r = append(r, f(v))
	}
	return r
}

// Filter returns a new slice containing all elements of the slice that satisfy the predicate function.
func Filter[T any](s []T, f func(T) bool) []T {
	r := make([]T, 0, len(s))
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

func First[T any](s []T) T {
	return s[0]
}

func Sum[T int | float64](s ...T) T {
	var v T
	for _, x := range s {
		v += x
	}
	return v
}
