package goboorulib

type Autocomplete struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	Value     string `json:"value"`
	PostCount int    `json:"post_count,string"`
	Category  string `json:"category"`
}
