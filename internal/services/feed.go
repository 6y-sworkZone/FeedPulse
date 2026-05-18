package services

import (
	"strings"
	"time"

	"feedpulse/internal/database"
	"feedpulse/internal/models"
	"feedpulse/internal/utils"
)

func AddFeed(userID int64, feedURL string) (*models.Feed, error) {
	parsedFeed, err := utils.FetchFeed(feedURL)
	if err != nil {
		return nil, err
	}

	result, err := database.DB.Exec(
		`INSERT INTO feeds (user_id, url, title, description, site_url, fetch_interval)
		 VALUES (?, ?, ?, ?, ?, 30)`,
		userID, feedURL, parsedFeed.Title, parsedFeed.Description, parsedFeed.Link,
	)
	if err != nil {
		return nil, err
	}

	feedID, err := result.LastInsertId()
	if err != nil {
		return nil, err
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

		database.DB.Exec(
			`INSERT OR IGNORE INTO articles (feed_id, user_id, guid, title, url, summary, content, author, published_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			feedID, userID, guid, item.Title, item.Link, item.Description, item.Content,
			getAuthor(item), published,
		)
	}

	return GetFeedByID(feedID)
}

func getAuthor(item interface{}) string {
	if i, ok := item.(*struct{ Author string }); ok {
		return i.Author
	}
	return ""
}

func GetFeedByID(feedID int64) (*models.Feed, error) {
	var feed models.Feed
	err := database.DB.QueryRow(
		`SELECT id, user_id, url, title, description, site_url, favicon_url,
		 health_status, failure_count, last_fetched_at, fetch_interval, created_at, updated_at
		 FROM feeds WHERE id = ?`,
		feedID,
	).Scan(&feed.ID, &feed.UserID, &feed.URL, &feed.Title, &feed.Description, &feed.SiteURL,
		&feed.FaviconURL, &feed.HealthStatus, &feed.FailureCount, &feed.LastFetchedAt,
		&feed.FetchInterval, &feed.CreatedAt, &feed.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

func GetUserFeeds(userID int64) ([]models.Feed, error) {
	rows, err := database.DB.Query(
		`SELECT id, user_id, url, title, description, site_url, favicon_url,
		 health_status, failure_count, last_fetched_at, fetch_interval, created_at, updated_at
		 FROM feeds WHERE user_id = ? ORDER BY title`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var feed models.Feed
		err := rows.Scan(&feed.ID, &feed.UserID, &feed.URL, &feed.Title, &feed.Description,
			&feed.SiteURL, &feed.FaviconURL, &feed.HealthStatus, &feed.FailureCount,
			&feed.LastFetchedAt, &feed.FetchInterval, &feed.CreatedAt, &feed.UpdatedAt)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, nil
}

func DeleteFeed(userID, feedID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM feeds WHERE id = ? AND user_id = ?",
		feedID, userID,
	)
	return err
}

func UpdateFeedHealth(feedID int64, success bool) error {
	if success {
		_, err := database.DB.Exec(
			"UPDATE feeds SET health_status = 'healthy', failure_count = 0, last_fetched_at = ? WHERE id = ?",
			time.Now(), feedID,
		)
		return err
	}

	var failureCount int
	err := database.DB.QueryRow("SELECT failure_count FROM feeds WHERE id = ?", feedID).Scan(&failureCount)
	if err != nil {
		return err
	}

	failureCount++
	status := "healthy"
	if failureCount >= 3 {
		status = "unhealthy"
	}

	_, err = database.DB.Exec(
		"UPDATE feeds SET health_status = ?, failure_count = ?, last_fetched_at = ? WHERE id = ?",
		status, failureCount, time.Now(), feedID,
	)
	return err
}

func CreateGroup(userID int64, name string) (*models.Group, error) {
	result, err := database.DB.Exec(
		"INSERT INTO groups (user_id, name) VALUES (?, ?)",
		userID, name,
	)
	if err != nil {
		return nil, err
	}

	groupID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return GetGroupByID(groupID)
}

func GetGroupByID(groupID int64) (*models.Group, error) {
	var group models.Group
	err := database.DB.QueryRow(
		"SELECT id, user_id, name, sort_order, created_at, updated_at FROM groups WHERE id = ?",
		groupID,
	).Scan(&group.ID, &group.UserID, &group.Name, &group.SortOrder, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func GetUserGroups(userID int64) ([]models.Group, error) {
	rows, err := database.DB.Query(
		"SELECT id, user_id, name, sort_order, created_at, updated_at FROM groups WHERE user_id = ? ORDER BY sort_order, name",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var group models.Group
		err := rows.Scan(&group.ID, &group.UserID, &group.Name, &group.SortOrder,
			&group.CreatedAt, &group.UpdatedAt)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func AddFeedToGroup(feedID, groupID int64) error {
	_, err := database.DB.Exec(
		"INSERT OR IGNORE INTO feed_groups (feed_id, group_id) VALUES (?, ?)",
		feedID, groupID,
	)
	return err
}

func RemoveFeedFromGroup(feedID, groupID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM feed_groups WHERE feed_id = ? AND group_id = ?",
		feedID, groupID,
	)
	return err
}

func DeleteGroup(userID, groupID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM groups WHERE id = ? AND user_id = ?",
		groupID, userID,
	)
	return err
}

func ImportOPML(userID int64, opmlData []byte) error {
	opml, err := utils.ParseOPML(strings.NewReader(string(opmlData)))
	if err != nil {
		return err
	}

	feeds := utils.ExtractFeedURLs(opml)
	for _, feed := range feeds {
		_, err := AddFeed(userID, feed.URL)
		if err != nil {
			continue
		}
	}
	return nil
}

func ExportOPML(userID int64) ([]byte, error) {
	feeds, err := GetUserFeeds(userID)
	if err != nil {
		return nil, err
	}

	feedList := make([]struct{ Title, XMLURL, HTMLURL string }, len(feeds))
	for i, f := range feeds {
		feedList[i] = struct {
			Title   string
			XMLURL  string
			HTMLURL string
		}{Title: f.Title, XMLURL: f.URL, HTMLURL: f.SiteURL}
	}

	return utils.GenerateOPML(feedList)
}
