package newtqnia

import "time"

// Locale is a supported response language.
type Locale string

const (
	LocaleEnglish Locale = "en"
	LocaleArabic  Locale = "ar"
)

// Collection identifies a news collection.
type Collection string

const (
	CollectionToday  Collection = "today"
	CollectionLatest Collection = "latest"
)

// LinkLabel contains a display label and its canonical URL.
type LinkLabel struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Attribution contains the attribution that clients must display with API content.
type Attribution struct {
	Text     string `json:"text"`
	URL      string `json:"url"`
	Required bool   `json:"required"`
}

// Category identifies an article category.
type Category struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Article is one localized NewTqnia article summary.
type Article struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Category    Category  `json:"category"`
	Image       string    `json:"image"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	ReadTime    int       `json:"read_time"`
}

// Digest is a localized collection of news and its attribution metadata.
type Digest struct {
	APIVersion  string            `json:"api_version"`
	Collection  Collection        `json:"collection"`
	Date        string            `json:"date,omitempty"`
	Timezone    string            `json:"timezone"`
	Locale      Locale            `json:"locale"`
	Direction   string            `json:"direction"`
	Publisher   LinkLabel         `json:"publisher"`
	Attribution Attribution       `json:"attribution"`
	GeneratedAt time.Time         `json:"generated_at"`
	Articles    []Article         `json:"articles"`
	Links       map[string]string `json:"_links"`
}

// NewsListParams filters either public news collection.
type NewsListParams struct {
	// Locale defaults to LocaleEnglish.
	Locale Locale
	// Limit defaults to 10 and must be between 1 and 10 when set.
	Limit int
	// Category is an optional category slug.
	Category string
}
