package gobooru

// Library specific struct
type Post struct {
	ID          int64    `json:"id"`
	SourceBooru string   `json:"source_booru"`
	FileUrl     string   `json:"file_url"`
	PreviewUrl  string   `json:"preview_url"`
	Tags        []string `json:"tags"`
	Score       int      `json:"score"`
	Rating      string   `json:"rating"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	MD5         string   `json:"md5"`
}
