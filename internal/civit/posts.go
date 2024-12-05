package civit

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/carlmjohnson/requests"
)

type Posts struct {
	client *Client
}

func NewPosts(client *Client) *Posts {
	return &Posts{client: client}
}

func (p *Posts) Download(ids ...int) error {
	ctx := context.Background()
	for _, id := range ids {
		var resp struct {
			Items []*Image
		}

		if err := p.client.Request().Path("/api/v1/images").Param("nsfw", "X").ParamInt("postId", id).ToJSON(&resp).Fetch(ctx); err != nil {
			return err
		}
		fmt.Printf("Downloaded %d items\n", len(resp.Items))
		for _, item := range resp.Items {
			fmt.Println(item.Url)
			if err := os.MkdirAll(fmt.Sprintf("posts/%d", id), 0755); err != nil {
				return err
			}
			if err := requests.URL(item.Url).ToFile(fmt.Sprintf("posts/%d/%s", id, path.Base(item.Url))).Fetch(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}
