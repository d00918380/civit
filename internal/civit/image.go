package civit

import "time"

type Item struct {
	Id            int       `json:"id"`
	Url           string    `json:"url"`
	Hash          string    `json:"hash"`
	Widht         int       `json:"width"`
	Height        int       `json:"height"`
	NSFWLevel     string    `json:"nsfwLevel"`
	NSFW          bool      `json:"nsfw"`
	BrowsingLevel int       `json:"browsingLevel"`
	CreatedAt     time.Time `json:"createdAt"`
	PostId        int       `json:"postId"`
	Index         int       `json:"index"`
	PublishedAt   time.Time `json:"publishedAt"`
	Stats         ItemStats `json:"stats"`
	Meta          *ItemMeta `json:"meta,omitempty"`
	Username      string    `json:"username"`
	BaseModel     string    `json:"baseModel"`
}

type ItemStats struct {
	CryCount     int `json:"cryCount"`
	LaughCount   int `json:"laughCount"`
	LikeCount    int `json:"likeCount"`
	DislikeCount int `json:"dislikeCount"`
	HeartCount   int `json:"heartCount"`
	CommentCount int `json:"commentCount"`
}

type ItemMeta struct {
	Size             string          `json:"Size"`
	Seed             int             `json:"seed"`
	Extra            any             `json:"extra,omitempty"`
	Steps            int             `json:"steps"`
	Prompt           string          `json:"prompt,omitempty"`
	Sampler          string          `json:"sampler"`
	CfgScale         float64         `json:"cfgScale"`
	ClipSkip         int             `json:"clipSkip"`
	Resources        []any           `json:"resources,omitempty"`
	CreatedDate      string          `json:"Created Date"`
	NegativePrompt   string          `json:"negativePrompt,omitempty"`
	CivitaiResources []CivitResource `json:"civitaiResources,omitempty"`
}

type CivitResource struct {
	Type             string  `json:"type"`
	Weight           float64 `json:"weight"`
	ModelVersionId   int     `json:"modelVersionId"`
	ModelVersionName string  `json:"modelVersionName"`
}

type Metadata struct {
	NextPage string `json:"nextPage"`
}
