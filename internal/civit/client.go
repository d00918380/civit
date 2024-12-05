package civit

import (
	"fmt"

	"github.com/carlmjohnson/requests"
)

type Client struct {
	apiKey string
}

func NewClient(key string) *Client {
	return &Client{apiKey: key}
}

func (c *Client) Request() *requests.Builder {
	return requests.
		URL("https://civitai.com").
		Header("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
}
