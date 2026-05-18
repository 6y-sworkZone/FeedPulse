package models

import (
	"time"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserSettings struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	Theme        string `json:"theme"`
	FontSize     string `json:"font_size"`
	ArticlesPerPage int `json:"articles_per_page"`
	ReadingPreferences string `json:"reading_preferences"`
}

type Feed struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	URL             string    `json:"url"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	SiteURL         string    `json:"site_url"`
	FaviconURL      string    `json:"favicon_url"`
	HealthStatus    string    `json:"health_status"`
	FailureCount    int       `json:"failure_count"`
	LastFetchedAt   *time.Time `json:"last_fetched_at"`
	FetchInterval   int       `json:"fetch_interval"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Group struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FeedGroup struct {
	FeedID  int64 `json:"feed_id"`
	GroupID int64 `json:"group_id"`
}

type Article struct {
	ID          int64     `json:"id"`
	FeedID      int64     `json:"feed_id"`
	UserID      int64     `json:"user_id"`
	GUID        string    `json:"guid"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	IsRead      bool      `json:"is_read"`
	IsStarred   bool      `json:"is_starred"`
	IsLater     bool      `json:"is_later"`
	ReadAt      *time.Time `json:"read_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type WebhookConfig struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	URL         string    `json:"url"`
	Enabled     bool      `json:"enabled"`
	Frequency   string    `json:"frequency"`
	Keywords    string    `json:"keywords"`
	FeedIDs     string    `json:"feed_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Notification struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ArticleID   *int64    `json:"article_id"`
	SentAt      time.Time `json:"sent_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type ReadingStats struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Date        time.Time `json:"date"`
	Hour        int       `json:"hour"`
	Weekday     int       `json:"weekday"`
	ReadCount   int       `json:"read_count"`
}
