package trpc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/carlmjohnson/requests"
	"go.nhat.io/cookiejar"
	"golang.org/x/net/publicsuffix"
)

// Client is TRPC client for the Civit API.
type Client struct {
	client *http.Client
}

type CursorResult[T any] struct {
	Data struct {
		JSON struct {
			Items      []T    `json:"items"`
			NextCursor string `json:"nextCursor"`
		} `json:"json"`
	} `json:"data"`
}

type CursorIterator[T any] struct {
	client *http.Client
	ctx    context.Context
	items  []T
	nextFn func(string) string
	err    error
	url    string
	token  string
}

func (i *CursorIterator[T]) Next() bool {
	if i.err != nil {
		return false
	}
	if len(i.items) > 0 {
		return true
	}
	var response struct {
		CursorResult[T] `json:"result"`
	}
	if i.url == "" {
		return false
	}
	if err := requests.URL(i.url).Client(i.client).ToJSON(&response).Fetch(i.ctx); err != nil {
		i.err = err
		return false
	}
	i.items = response.CursorResult.Data.JSON.Items
	switch response.Data.JSON.NextCursor {
	case "":
		i.url = ""
	default:
		i.url = i.nextFn(response.Data.JSON.NextCursor)
	}
	return len(i.items) > 0
}

func (i *CursorIterator[T]) Item() T {
	return pop(&i.items)
}

// pop removes the first element from the slice
// and returns it.
func pop[T any](s *[]T) T {
	v := (*s)[0]
	*s = (*s)[1:]
	return v
}
func (i *CursorIterator[T]) Err() error {
	return i.err
}

// New creates a new Civit TRCP API client.
func New(token string, cookiesfile string) *Client {
	jar := cookiejar.NewPersistentJar(
		cookiejar.WithFilePath(cookiesfile),
		cookiejar.WithAutoSync(true),
		// All users of cookiejar should import "golang.org/x/net/publicsuffix"
		cookiejar.WithPublicSuffixList(publicsuffix.List),
	)
	client := *http.DefaultClient
	client.Jar = jar
	return &Client{client: &client}
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

func (c *Client) QueryGeneratedImages(ctx context.Context) *CursorIterator[GeneratedItem] {
	iter := &CursorIterator[GeneratedItem]{ctx: ctx, client: c.client, nextFn: func(cursor string) string {
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

type Item struct {
	ID       int    `json:"id"`
	Index    int    `json:"index"`
	PostID   int    `json:"postId"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Hash     string `json:"hash"`
	HideMeta bool   `json:"hideMeta"`
	HasMeta  bool   `json:"hasMeta"`
	OnSite   bool   `json:"onSite"`
	// CreatedAt   time.Time `json:"createdAt"`
	// SortAt      time.Time `json:"sortAt"`
	PublishedAt time.Time `json:"publishedAt"`
	// ReactionCount int       `json:"reactionCount"`
	Type  string `json:"type"`
	Stats struct {
		LikeCountAllTime         int `json:"likeCountAllTime"`
		LaughCountAllTime        int `json:"laughCountAllTime"`
		HeartCountAllTime        int `json:"heartCountAllTime"`
		CryCountAllTime          int `json:"cryCountAllTime"`
		CommentCountAllTime      int `json:"commentCountAllTime"`
		CollectedCountAllTime    int `json:"collectedCountAllTime"`
		TippedAmountCountAllTime int `json:"tippedAmountCountAllTime"`
		// DislikeCountAllTime      int `json:"dislikeCountAllTime"`
		// ViewCountAllTime         int `json:"viewCountAllTime"`
	} `json:"stats"`
	User struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
	// Availability string `json:"availability"`
}

func (i *Item) Published() bool {
	return !i.PublishedAt.IsZero()
}

func (c *Client) AddImageToShowcase(ctx context.Context, id int) error {
	return requests.URL("https://civitai.com/api/trpc/userProfile.addEntityToShowcase").Client(c.client).BodyJSON(map[string]any{
		"json": map[string]any{
			"entityId":   id,
			"entityType": "Image",
			"authed":     true,
		}}).Post().Fetch(ctx)
}

func (c *Client) ImagesForPost(ctx context.Context, id int) *CursorIterator[Item] {
	iter := &CursorIterator[Item]{ctx: ctx, client: c.client, nextFn: func(cursor string) string {
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

func (c *Client) ImagesForUsername(ctx context.Context, username string) *CursorIterator[Item] {
	iter := &CursorIterator[Item]{ctx: ctx, client: c.client, nextFn: func(cursor string) string {
		switch cursor {
		case "":
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"username":"`+username+`","useIndex":true,"browsingLevel":31,"cursor":null,"authed":true},"meta":{"values":{"cursor":["undefined"]}}}`))
		default:
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"username":"`+username+`","useIndex":true,"browsingLevel":31,"cursor":"`+cursor+`","authed":true}}`))
		}
	}}
	iter.url = iter.nextFn("")
	return iter
}

func (c *Client) ImagesForUser(ctx context.Context, username string, id int) *CursorIterator[Item] {
	iter := &CursorIterator[Item]{ctx: ctx, client: c.client, nextFn: func(cursor string) string {
		switch cursor {
		case "":
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"period":"AllTime","sort":"Newest","types":["image"],"username":"`+username+`","withMeta":false,"fromPlatform":false,"userId":`+strconv.Itoa(id)+`,"useIndex":true,"browsingLevel":31,"include":["cosmetics"],"cursor":null,"authed":true},"meta":{"values":{"cursor":["undefined"]}}}`))
		default:
			return fmt.Sprintf("https://civitai.com/api/trpc/image.getInfinite?input=%s", url.QueryEscape(`{"json":{"period":"AllTime","sort":"Newest","types":["image"],"username":"`+username+`","withMeta":false,"fromPlatform":false,"userId":`+strconv.Itoa(id)+`,"useIndex":true,"browsingLevel":31,"include":["cosmetics"],"cursor":"`+cursor+`","authed":true}}`))
		}
	}}
	iter.url = iter.nextFn("")
	return iter
}

func (c *Client) Image(ctx context.Context, id int) (*Item, error) {
	var response struct {
		Result struct {
			Data struct {
				Item `json:"json"`
			} `json:"data"`
		} `json:"result"`
	}
	url := fmt.Sprintf("https://civitai.com/api/trpc/image.get?input=%s", url.QueryEscape(fmt.Sprintf(`{"json":{"id":%d,"authed":true}}`, id)))
	return &response.Result.Data.Item, requests.URL(url).Client(c.client).ToJSON(&response).Fetch(ctx)
}

type Model struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	ModelVersions []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Rank struct {
			GenerationCountAllTime int `json:"generationCountAllTime"`
			DownloadCountAllTime   int `json:"downloadCountAllTime"`
			RatingCountAllTime     int `json:"ratingCountAllTime"`
			RatingAllTime          int `json:"ratingAllTime"`
			ThumbsUpCountAllTime   int `json:"thumbsUpCountAllTime"`
			ThumbsDownCountAllTime int `json:"thumbsDownCountAllTime"`
		} `json:"rank"`
	} `json:"modelVersions"`
}

func (c *Client) Model(ctx context.Context, id int) (*Model, error) {
	var response struct {
		Result struct {
			Data struct {
				Model `json:"json"`
			} `json:"data"`
		} `json:"result"`
	}
	url := fmt.Sprintf("https://civitai.com/api/trpc/model.getById?input=%s", url.QueryEscape(fmt.Sprintf(`{"json":{"id":%d,"authed":true}}`, id)))
	return &response.Result.Data.Model, requests.URL(url).Client(c.client).ToJSON(&response).Fetch(ctx)
}

type CompensationPool struct {
	// Value is the value of the current compensation pool in US dollars.
	Value float64 `json:"value"`
	Size  struct {
		// Current is the current size of banked buzz.
		Current float64 `json:"current"`
		// Forecasted is the forecasted size of banked buzz.
		Forecasted float64 `json:"forecasted"`
	} `json:"size"`
}

func (c *Client) CreatorProgramGetCompensationPool(ctx context.Context) (*CompensationPool, error) {
	var response struct {
		Result struct {
			Data struct {
				CompensationPool `json:"json"`
			} `json:"data"`
		} `json:"result"`
	}
	url := `https://civitai.com/api/trpc/creatorProgram.getCompensationPool?input=%7B%22json%22%3A%7B%22authed%22%3Atrue%7D%7D`
	return &response.Result.Data.CompensationPool, requests.URL(url).Client(c.client).ToJSON(&response).Fetch(ctx)
}

type Result[T any] struct {
	Data struct {
		JSON []T `json:"json"`
	} `json:"data"`
}

type Iterator[T any] struct {
	ctx   context.Context
	items []T
	err   error
	url   string
	token string
}

func (i *Iterator[T]) Next() bool {
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
	fmt.Println(i.url)
	if err := requests.URL(i.url).Header("Authorization", "Bearer "+i.token).ToJSON(&response).Fetch(i.ctx); err != nil {
		i.err = err
		return false
	}
	i.items = response.Result.Data.JSON
	return len(i.items) > 0
}

func (i *Iterator[T]) Item() T {
	return pop(&i.items)
}

func (i *Iterator[T]) Err() error {
	return i.err
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func (c *Client) UsersFollowing(ctx context.Context) *CursorIterator[User] {
	iter := &CursorIterator[User]{ctx: ctx, client: c.client, nextFn: func(_ string) string {
		return fmt.Sprintf("https://civitai.com/api/trpc/user.getFollowingUsers?input=%s", url.QueryEscape(`{"json":{"authed":true}}`))
	}}
	iter.url = iter.nextFn("")
	return iter
}

type Lists struct {
	Following []User `json:"following"`
	Followers []User `json:"followers"`
}

func (c *Client) ListsForUser(ctx context.Context, username string) (*Lists, error) {
	var response struct {
		Result struct {
			Data struct {
				Lists `json:"json"`
			} `json:"data"`
		} `json:"result"`
	}
	url := fmt.Sprintf("https://civitai.com/api/trpc/user.getLists?input=%s", url.QueryEscape(`{"json":{"username":"`+username+`"}}`))
	return &response.Result.Data.Lists, requests.URL(url).Client(c.client).ToJSON(&response).Fetch(ctx)
}
