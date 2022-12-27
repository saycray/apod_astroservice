package astro

type AstroModel struct {
	Date        string `json:"date" db:"date"`
	Title       string `json:"title" db:"title"`
	URL         string `json:"url" db:"url"`
	HDURL       string `json:"hd_url" db:"hd_url"`
	ThumbURL    string `json:"thumbnail_url" db:"thumbnail_url"`
	MediaType   string `json:"media_type" db:"media_type"`
	Copyright   string `json:"copyright" db:"copyright"`
	Explanation string `json:"explanation" db:"explanation"`
	RAW         []byte `json:"raw" db:"-"`
}
