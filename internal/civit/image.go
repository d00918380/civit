package civit

type Image struct {
	Id        int    `json:"id"`
	CreatedAt string `json:"createdAt"`
	Url       string `json:"url"`
	PostId    int    `json:"postId"`
	Username  string `json:"username"`
}

type Metadata struct {
	NextPage string `json:"nextPage"`
}
