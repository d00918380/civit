package trpc

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/carlmjohnson/requests"
)

// Client is TRPC client for the Civit API.
type Client struct {
	token string
}

type Result[T any] struct {
	Data Data[T] `json:"data"`
}

type Data[T any] struct {
	JSON JSON[T] `json:"json"`
}

type JSON[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor"`
}

type Iter[T any] struct {
	ctx    context.Context
	items  []T
	nextFn func(string) string
	err    error
	url    string
	token  string
}

func (i *Iter[T]) Next() bool {
	if i.err != nil {
		return false
	}
	if len(i.items) > 0 {
		return true
	}
	var response struct {
		Result[T] `json:"result"`
	}
	if i.url == "" {
		return false
	}
	if err := requests.URL(i.url).Header("Authorization", "Bearer "+i.token).ToJSON(&response).Fetch(i.ctx); err != nil {
		i.err = err
		return false
	}
	i.items = response.Result.Data.JSON.Items
	switch response.Data.JSON.NextCursor {
	case "":
		i.url = ""
	default:
		i.url = i.nextFn(response.Data.JSON.NextCursor)
	}
	return len(i.items) > 0
}

func (i *Iter[T]) Item() T {
	return pop(&i.items)
}

// pop removes the first element from the slice
// and returns it.
func pop[T any](s *[]T) T {
	v := (*s)[0]
	*s = (*s)[1:]
	return v
}
func (i *Iter[T]) Err() error {
	return i.err
}

// New creates a new Civit TRCP API client.
func New(token string) *Client {
	return &Client{token: token}
}

// GeneratedItem is a generated image.
type GeneratedItem struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Steps     []struct {
		Images []struct {
			Type      string    `json:"type"`
			ID        string    `json:"id"`
			Completed time.Time `json:"completed"`
			URL       string    `json:"url"`
			Widht     int       `json:"width"`
			Height    int       `json:"height"`
		} `json:"images"`
	} `json:"steps"`
}

func (c *Client) QueryGeneratedImages(ctx context.Context) *Iter[GeneratedItem] {
	iter := &Iter[GeneratedItem]{ctx: ctx, token: c.token, nextFn: func(cursor string) string {
		switch cursor {
		case "":
			return fmt.Sprintf("https://civitai.com/api/trpc/orchestrator.queryGeneratedImages?input=%s", url.QueryEscape(`{"json":{"tags":["gen"],"cursor":null,"authed":true},"meta":{"values":{"cursor":["undefined"]}}}`))
		default:
			return fmt.Sprintf("https://civitai.com/api/trpc/orchestrator.queryGeneratedImages?input=%s", url.QueryEscape(`{"json":{"tags":["gen"],"cursor":"`+cursor+`","authed":true}}`))
		}
	}}
	iter.url = iter.nextFn("")
	return iter
}

type Image struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	NSFWLevel int       `json:"nsfwLevel"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Hash      string    `json:"hash"`
	HideMeta  bool      `json:"hideMeta"`
	HasMeta   bool      `json:"hasMeta"`
	OnSite    bool      `json:"onSite"`
	CreatedAt time.Time `json:"createdAt"`
	SortAt    time.Time `json:"sortAt"`
	MimeType  string    `json:"mimeType"`
	Type      string    `json:"type"`
	PostID    int       `json:"postId"`
}

func (c *Client) ImagesForPost(ctx context.Context, id int) *Iter[Image] {
	iter := &Iter[Image]{ctx: ctx, token: c.token, nextFn: func(cursor string) string {
		switch cursor {
		case "":
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"postId":`+strconv.Itoa(id)+`,"pending":true,"browsingLevel":null,"cursor":null,"authed":true},"meta":{"values":{"browsingLevel":["undefined"],"cursor":["undefined"]}}}`))
		default:
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"postId":`+strconv.Itoa(id)+`,"pending":true,"browsingLevel":null,"cursor":"`+cursor+`","authed":true}}`))
		}
	}}
	iter.url = iter.nextFn("")
	return iter
}

func (c *Client) ImagesForUser(ctx context.Context, id int) *Iter[Image] {
	iter := &Iter[Image]{ctx: ctx, token: c.token, nextFn: func(cursor string) string {
		switch cursor {
		case "":
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"userId":`+strconv.Itoa(id)+`,"pending":true,"browsingLevel":null,"cursor":null,"authed":true},"meta":{"values":{"browsingLevel":["undefined"],"cursor":["undefined"]}}}`))
		default:
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"userId":`+strconv.Itoa(id)+`,"pending":true,"browsingLevel":null,"cursor":"`+cursor+`","authed":true}}`))
		}
	}}
	iter.url = iter.nextFn("")
	return iter
}
