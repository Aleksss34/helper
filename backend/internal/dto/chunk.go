package dto

type Chunk struct {
	Server       string `json:"server"`
	ArticleTitle string `json:"article_title"`
	SectionTitle string `json:"section_title,omitempty"`
	SourceURL    string `json:"source_url"`
	Text         string `json:"text"`
}

type Article struct {
	Title   string
	Server  string
	URL     string
	Content string
}
