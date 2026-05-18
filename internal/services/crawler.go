package services

import (
	"log"
	"time"

	"feedpulse/internal/database"
	"feedpulse/internal/utils"
)

func CrawlAllFeeds() {
	log.Println("Starting feed crawl...")

	rows, err := database.DB.Query("SELECT id, url FROM feeds")
	if err != nil {
		log.Printf("Error getting feeds: %v", err)
		return
	}
	defer rows.Close()

	type feedToCrawl struct {
		ID  int64
		URL string
	}

	var feeds []feedToCrawl
	for rows.Next() {
		var f feedToCrawl
		if err := rows.Scan(&f.ID, &f.URL); err == nil {
			feeds = append(feeds, f)
		}
	}

	for _, feed := range feeds {
		if err := CrawlFeed(feed.ID, feed.URL); err != nil {
			log.Printf("Error crawling feed %d: %v", feed.ID, err)
		}
	}

	log.Println("Feed crawl completed")
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

		_, err := database.DB.Exec(
			`INSERT OR IGNORE INTO articles (feed_id, user_id, guid, title, url, summary, content, author, published_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, userID, guid, item.Title, item.Link, item.Description, content,
			author, published,
		)
		if err != nil {
			log.Printf("Error inserting article: %v", err)
		}
	}

	return nil
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
