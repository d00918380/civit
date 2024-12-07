package civit

import (
	"context"
	"strconv"

	"github.com/carlmjohnson/requests"
)

func New(token string) *Client {
	return &Client{token: token}
}

type Client struct {
	token string
}

func (c *Client) ItemsForUser(ctx context.Context, username string) ([]*Item, error) {
	url := "https://civitai.com/api/v1/images?nsfw=X&token=" + c.token + "&username=" + username
	return fetchItems(ctx, url)
}

func (c *Client) ItemsForPost(ctx context.Context, postID int) ([]*Item, error) {
	url := "https://civitai.com/api/v1/images?nsfw=X&token=" + c.token + "&postId=" + strconv.Itoa(postID)
	return fetchItems(ctx, url)
}

func fetchItems(ctx context.Context, url string) ([]*Item, error) {
	var resp struct {
		Items    []*Item  `json:"items"`
		Metadata Metadata `json:"metadata"`
	}
	err := requests.URL(url).ToJSON(&resp).Fetch(ctx)
	if err != nil {
		return nil, err
	}
	items := resp.Items
	if resp.Metadata.NextPage != "" { // another page?
		nextItems, err := fetchItems(ctx, resp.Metadata.NextPage)
		if err != nil {
			return nil, err
		}
		items = append(items, nextItems...)
	}
	return items, nil
}
