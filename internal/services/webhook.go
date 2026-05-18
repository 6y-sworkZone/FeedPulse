package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"feedpulse/internal/database"
	"feedpulse/internal/models"
)

func GetWebhookConfigs(userID int64) ([]models.WebhookConfig, error) {
	rows, err := database.DB.Query(
		`SELECT id, user_id, url, enabled, frequency, keywords, feed_ids, created_at, updated_at
		 FROM webhook_configs WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []models.WebhookConfig
	for rows.Next() {
		var c models.WebhookConfig
		err := rows.Scan(&c.ID, &c.UserID, &c.URL, &c.Enabled, &c.Frequency,
			&c.Keywords, &c.FeedIDs, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func CreateWebhookConfig(userID int64, url string, enabled bool, frequency, keywords, feedIDs string) (*models.WebhookConfig, error) {
	result, err := database.DB.Exec(
		`INSERT INTO webhook_configs (user_id, url, enabled, frequency, keywords, feed_ids)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, url, enabled, frequency, keywords, feedIDs,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	var config models.WebhookConfig
	err = database.DB.QueryRow(
		`SELECT id, user_id, url, enabled, frequency, keywords, feed_ids, created_at, updated_at
		 FROM webhook_configs WHERE id = ?`,
		id,
	).Scan(&config.ID, &config.UserID, &config.URL, &config.Enabled, &config.Frequency,
		&config.Keywords, &config.FeedIDs, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func DeleteWebhookConfig(userID, configID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM webhook_configs WHERE id = ? AND user_id = ?",
		configID, userID,
	)
	return err
}

func SendWebhookNotification(config models.WebhookConfig, article models.Article) error {
	payload := map[string]interface{}{
		"type": "new_article",
		"article": map[string]interface{}{
			"id":          article.ID,
			"title":       article.Title,
			"url":         article.URL,
			"summary":     article.Summary,
			"author":      article.Author,
			"published_at": article.PublishedAt,
		},
		"timestamp": time.Now(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(config.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = database.DB.Exec(
		`INSERT INTO notifications (user_id, type, title, content, article_id)
		 VALUES (?, ?, ?, ?, ?)`,
		config.UserID, "webhook", article.Title, config.URL, article.ID,
	)

	return err
}

func ProcessNewArticleNotifications(article models.Article) {
	configs, err := GetWebhookConfigs(article.UserID)
	if err != nil {
		return
	}

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		if config.Frequency != "realtime" {
			continue
		}

		if config.Keywords != "" {
			keywords := splitKeywords(config.Keywords)
			found := false
			for _, k := range keywords {
				if containsKeyword(article.Title, k) || containsKeyword(article.Summary, k) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if config.FeedIDs != "" {
			allowedFeeds := splitFeedIDs(config.FeedIDs)
			found := false
			for _, fid := range allowedFeeds {
				if fid == article.FeedID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		SendWebhookNotification(config, article)
	}
}

func splitKeywords(s string) []string {
	var result []string
	for _, k := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(k); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitFeedIDs(s string) []int64 {
	var result []int64
	for _, f := range strings.Split(s, ",") {
		var id int64
		if _, err := fmt.Sscanf(f, "%d", &id); err == nil {
			result = append(result, id)
		}
	}
	return result
}

func containsKeyword(s, keyword string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(keyword))
}

func ProcessHourlyWebhookDigest() {
	processWebhookDigest("hourly")
}

func ProcessDailyWebhookDigest() {
	processWebhookDigest("daily")
}

func processWebhookDigest(frequency string) {
	rows, err := database.DB.Query(
		`SELECT DISTINCT user_id FROM webhook_configs WHERE enabled = 1 AND frequency = ?`,
		frequency,
	)
	if err != nil {
		log.Printf("Error getting webhook configs: %v", err)
		return
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err == nil {
			userIDs = append(userIDs, userID)
		}
	}

	for _, userID := range userIDs {
		sendWebhookDigestForUser(userID, frequency)
	}
}

func sendWebhookDigestForUser(userID int64, frequency string) {
	configRows, err := database.DB.Query(
		`SELECT id, url, keywords, feed_ids FROM webhook_configs 
		 WHERE user_id = ? AND enabled = 1 AND frequency = ?`,
		userID, frequency,
	)
	if err != nil {
		log.Printf("Error getting webhook configs for user %d: %v", userID, err)
		return
	}
	defer configRows.Close()

	var timeFilter string
	if frequency == "hourly" {
		timeFilter = "datetime('now', '-1 hour')"
	} else {
		timeFilter = "datetime('now', '-24 hours')"
	}

	for configRows.Next() {
		var configID int64
		var url, keywords, feedIDs string
		if err := configRows.Scan(&configID, &url, &keywords, &feedIDs); err != nil {
			continue
		}

		var articleArgs []interface{}
		articleQuery := `
			SELECT id, title, url, author, published_at, feed_id
			FROM articles 
			WHERE user_id = ? AND published_at >= ` + timeFilter

		articleArgs = append(articleArgs, userID)

		if feedIDs != "" {
			feedIDList := strings.Split(feedIDs, ",")
			placeholders := strings.Repeat("?,", len(feedIDList))
			articleQuery += " AND feed_id IN (" + placeholders[:len(placeholders)-1] + ")"
			for _, fid := range feedIDList {
				if id, err := strconv.ParseInt(strings.TrimSpace(fid), 10, 64); err == nil {
					articleArgs = append(articleArgs, id)
				}
			}
		}

		articleQuery += " ORDER BY published_at DESC"

		articleRows, err := database.DB.Query(articleQuery, articleArgs...)
		if err != nil {
			log.Printf("Error getting articles for digest: %v", err)
			continue
		}

		type ArticleDigest struct {
			ID          int64     `json:"id"`
			Title       string    `json:"title"`
			URL         string    `json:"url"`
			Author      string    `json:"author"`
			PublishedAt time.Time `json:"published_at"`
			FeedID      int64     `json:"feed_id"`
		}
		var articles []ArticleDigest

		for articleRows.Next() {
			var a ArticleDigest
			if err := articleRows.Scan(&a.ID, &a.Title, &a.URL, &a.Author, &a.PublishedAt, &a.FeedID); err == nil {
				if keywords != "" {
					keywordList := strings.Split(keywords, ",")
					matched := false
					for _, k := range keywordList {
						if containsKeyword(a.Title, k) {
							matched = true
							break
						}
					}
					if matched {
						articles = append(articles, a)
					}
				} else {
					articles = append(articles, a)
				}
			}
		}
		articleRows.Close()

		if len(articles) > 0 {
			payload := map[string]interface{}{
				"type":      frequency + "_digest",
				"articles":  articles,
				"count":     len(articles),
				"timestamp": time.Now(),
			}

			jsonData, err := json.Marshal(payload)
			if err != nil {
				continue
			}

			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err == nil {
				resp.Body.Close()
			}

			database.DB.Exec(
				`INSERT INTO notifications (user_id, type, title, content)
				 VALUES (?, ?, ?, ?)`,
				userID, "webhook_"+frequency, "Sent digest with "+strconv.Itoa(len(articles))+" articles", url,
			)
		}
	}
}
