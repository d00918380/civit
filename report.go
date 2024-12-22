package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"slices"
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
		"mean":       stats.Mean,
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
		"between": func(a, b time.Time) TimeRange {
			return TimeRange{a, b}
		},
		"worst_posts": func(r TimeRange) []*post {
			insideRange := func(t time.Time) bool {
				return t.After(r.Start) && t.Before(r.End)
			}

			posts := data.PostsByScore()
			posts = algorithms.Filter(posts, func(p *post) bool {
				return insideRange(p.CreatedAt())
			})
			slices.SortStableFunc(posts, func(a, b *post) int {
				return a.Score() - b.Score()
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
		ps = append(ps, p)
	}
	return ps
}

func (d *data) ImagesByScore() []*image {
	images := d.Images()
	slices.SortStableFunc(images, func(a, b *image) int {
		return b.Score() - a.Score()
	})
	return images
}

func (d *data) ImagesByDate() []*image {
	images := d.Images()
	slices.SortStableFunc(images, func(a, b *image) int {
		return int(b.CreatedAt.Sub(a.CreatedAt))
	})
	return images
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
		return int(b.CreatedAt().Sub(a.CreatedAt()))
	})
	return posts
}

func (d *data) PostScores() stats.Float64Data {
	return stats.Float64Data(Map(d.Posts(), func(p *post) float64 {
		return float64(p.Score())
	}))
}

func (d *data) ImageScores() stats.Float64Data {
	return stats.Float64Data(Map(d.Images(), func(i *image) float64 {
		return float64(i.Score())
	}))
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

// PostsJS prepares a fragment of javascript which represents the array of posts.
// Becauses its javascript, the final element in the array must not have a trailing comma. cool.
func (d *data) PostsJS() template.JS {
	var buf bytes.Buffer
	buf.WriteString("[\n")
	for i, p := range d.Posts() {
		buf.WriteString(fmt.Sprintf(`{id: %d, postURL: %q, score: %d, createdAt: new Date(%q)}`,
			p.Id(), p.PostURL(), p.Score(), p.CreatedAt().Format(time.RFC3339)))
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
			"createdAt": p.CreatedAt(),
		}
	})
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

func (p *post) CreatedAt() time.Time {
	return First(p.images).CreatedAt
}

func (p *post) Images() []*image {
	return p.images
}

func (p *post) Efficiency() float64 {
	return float64(p.Score()) / float64(len(p.Images()))
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

func Sum[T int](s ...T) T {
	var v T
	for _, x := range s {
		v += x
	}
	return v
}
