package main

import (
	"fmt"
	"html/template"
	"io"
	"slices"
	"time"

	"github.com/d00918380/civit/internal/civit"
)

const REPORT = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
	</head>
	<body>
		<title>Report</title>
		<ul>
			<li><a href="#posts_by_score">Posts by score</a></li>
			<li><a href="#posts_by_date">Posts by date</a></li>
			<li><a href="#posts_by_efficiency">Posts by efficiency</a></li>
			<li><a href="#images_by_score">Images by score</a></li>
			<li><a href="#images_by_date">Images by date</a></li>
		</ul>

		<h1 id="posts_by_score">Posts by score</h1>
		<ol>
			{{range .PostsByScore}}
				<li><a href="{{.PostURL}}">{{.PostURL}}</a>	Score: {{.Score}} Images: {{len .Images}} {{printf "%.2f" .Efficiency}}</li>
			{{end}}
		</ol>

		<h1 id="posts_by_date">Posts by date</h1>
		<ol>
			{{range .PostsByDate}}
				<li><a href="{{.PostURL}}">{{.PostURL}}</a>	Score: {{.Score}} Images: {{len .Images}} {{printf "%.2f" .Efficiency}}</li>
			{{end}}
		</ol>

		<h1 id="posts_by_efficiency">Posts by efficiency</h1>
		<ol>
			{{range .PostsByEfficiency}}
				<li><a href="{{.PostURL}}">{{.PostURL}}</a>	Score: {{.Score}} Images: {{len .Images}} {{printf "%.2f" .Efficiency}}</li>
			{{end}}
		</ol>

		<h1 id="images_by_score">Images by score</h1>
		<ol>
			{{range .ImagesByScore}}
				<li><a href="{{.ImageURL}}">{{.ImageURL}}</a> <a href="{{.PostURL}}">{{.PostURL}}</a> ({{.Score}})</li>
			{{end}}
		</ol>

		<h1 id="images_by_date">Images by date</h1>
		<ol>
			{{range .PostsByDate}}
			<li><a href="{{.PostURL}}">{{.PostURL}}</a> ({{.Score}})
			<ul>
				{{range .Images}}
					<li><a href="{{.ImageURL}}">{{.ImageURL}}</a> ({{.Score}})</li>
				{{end}}
			</ul>
			</li>
			{{end}}
		</ol>

	</body>
</html>`

func report(w io.Writer, items []*civit.Item) error {
	t, err := template.New("report").Parse(REPORT)
	if err != nil {
		return err
	}

	return t.Execute(w, &data{
		Items: items,
	})
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
