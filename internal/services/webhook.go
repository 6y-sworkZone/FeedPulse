package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
