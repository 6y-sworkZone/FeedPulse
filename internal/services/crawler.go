package services

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"feedpulse/internal/database"
	"feedpulse/internal/models"
	"feedpulse/internal/utils"
)

var lastCrawlTimes = make(map[int64]time.Time)

func CrawlAllFeeds() {
	log.Println("Starting scheduled feed crawl...")

	rows, err := database.DB.Query("SELECT id, url, fetch_interval, last_fetched_at FROM feeds")
	if err != nil {
		log.Printf("Error getting feeds: %v", err)
		return
	}
	defer rows.Close()

	type feedToCrawl struct {
		ID             int64
		URL            string
		FetchInterval  int
		LastFetchedAt  *time.Time
	}

	var feeds []feedToCrawl
	for rows.Next() {
		var f feedToCrawl
		if err := rows.Scan(&f.ID, &f.URL, &f.FetchInterval, &f.LastFetchedAt); err == nil {
			feeds = append(feeds, f)
		}
	}

	now := time.Now()
	for _, feed := range feeds {
		shouldCrawl := true
		if feed.LastFetchedAt != nil && feed.FetchInterval > 0 {
			nextCrawlTime := feed.LastFetchedAt.Add(time.Duration(feed.FetchInterval) * time.Minute)
			if now.Before(nextCrawlTime) {
				shouldCrawl = false
			}
		}

		if shouldCrawl {
			if err := CrawlFeed(feed.ID, feed.URL); err != nil {
				log.Printf("Error crawling feed %d: %v", feed.ID, err)
			}
		}
	}

	log.Println("Scheduled feed crawl completed")
}

func CrawlFeed(feedID int64, feedURL string) error {
	parsedFeed, err := utils.FetchFeed(feedURL)
	if err != nil {
		UpdateFeedHealth(feedID, false)
		return err
	}

	UpdateFeedHealth(feedID, true)

	var userID int64
	err = database.DB.QueryRow("SELECT user_id FROM feeds WHERE id = ?", feedID).Scan(&userID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, item := range parsedFeed.Items {
		published := now
		if item.PublishedParsed != nil {
			published = *item.PublishedParsed
		}

		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}

		author := ""
		if item.Author != nil {
			author = item.Author.Name
		}

		content := item.Content
		if content == "" {
			content = item.Description
		}

		summary := item.Description
		if summary == "" {
			summary = truncateString(content, 200)
		}

		var articleID int64
		err = database.DB.QueryRow(
			`SELECT id FROM articles WHERE user_id = ? AND guid = ?`,
			userID, guid,
		).Scan(&articleID)

		isNewArticle := err == sql.ErrNoRows

		if isNewArticle {
			if len(strings.TrimSpace(content)) < 200 && item.Link != "" {
				if fullContent, err := utils.ExtractFullContent(item.Link); err == nil {
					content = fullContent
				}
			}

			result, err := database.DB.Exec(
				`INSERT INTO articles (feed_id, user_id, guid, title, url, summary, content, author, published_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				feedID, userID, guid, item.Title, item.Link, summary, content,
				author, published,
			)
			if err != nil {
				log.Printf("Error inserting article: %v", err)
				continue
			}

			articleID, _ = result.LastInsertId()
			article := models.Article{
				ID:          articleID,
				FeedID:      feedID,
				UserID:      userID,
				GUID:        guid,
				Title:       item.Title,
				URL:         item.Link,
				Summary:     summary,
				Content:     content,
				Author:      author,
				PublishedAt: published,
			}
			ProcessNewArticleNotifications(article)
		}
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func GetPopularFeeds(limit int) ([]struct {
	Title string
	URL   string
	Count int
}, error) {
	rows, err := database.DB.Query(
		`SELECT title, site_url as url, COUNT(*) as count
		 FROM feeds
		 GROUP BY url
		 ORDER BY count DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []struct {
		Title string
		URL   string
		Count int
	}
	for rows.Next() {
		var f struct {
			Title string
			URL   string
			Count int
		}
		if err := rows.Scan(&f.Title, &f.URL, &f.Count); err == nil {
			feeds = append(feeds, f)
		}
	}
	return feeds, nil
}
