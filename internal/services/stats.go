package services

import (
	"feedpulse/internal/database"
)

type Stats struct {
	FeedCount     int `json:"feed_count"`
	ArticleCount  int `json:"article_count"`
	UnreadCount   int `json:"unread_count"`
	StarredCount  int `json:"starred_count"`
}

func GetUserStats(userID int64) (*Stats, error) {
	var stats Stats

	err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM feeds WHERE user_id = ?",
		userID,
	).Scan(&stats.FeedCount)
	if err != nil {
		return nil, err
	}

	err = database.DB.QueryRow(
		"SELECT COUNT(*) FROM articles WHERE user_id = ?",
		userID,
	).Scan(&stats.ArticleCount)
	if err != nil {
		return nil, err
	}

	err = database.DB.QueryRow(
		"SELECT COUNT(*) FROM articles WHERE user_id = ? AND is_read = 0",
		userID,
	).Scan(&stats.UnreadCount)
	if err != nil {
		return nil, err
	}

	err = database.DB.QueryRow(
		"SELECT COUNT(*) FROM articles WHERE user_id = ? AND is_starred = 1",
		userID,
	).Scan(&stats.StarredCount)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

type DailyStat struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func GetDailyReadingStats(userID int64, days int) ([]DailyStat, error) {
	rows, err := database.DB.Query(
		`SELECT date, SUM(read_count) as count
		 FROM reading_stats
		 WHERE user_id = ?
		 GROUP BY date
		 ORDER BY date DESC
		 LIMIT ?`,
		userID, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var s DailyStat
		if err := rows.Scan(&s.Date, &s.Count); err == nil {
			stats = append(stats, s)
		}
	}
	return stats, nil
}

type FeedArticleCount struct {
	FeedTitle string `json:"feed_title"`
	Count     int    `json:"count"`
}

func GetFeedArticleStats(userID int64) ([]FeedArticleCount, error) {
	rows, err := database.DB.Query(
		`SELECT f.title, COUNT(a.id) as count
		 FROM feeds f
		 LEFT JOIN articles a ON f.id = a.feed_id
		 WHERE f.user_id = ?
		 GROUP BY f.id
		 ORDER BY count DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []FeedArticleCount
	for rows.Next() {
		var s FeedArticleCount
		if err := rows.Scan(&s.FeedTitle, &s.Count); err == nil {
			stats = append(stats, s)
		}
	}
	return stats, nil
}

type HeatmapData struct {
	Hour     int `json:"hour"`
	Weekday  int `json:"weekday"`
	ReadCount int `json:"read_count"`
}

func GetReadingHeatmap(userID int64) ([]HeatmapData, error) {
	rows, err := database.DB.Query(
		`SELECT hour, weekday, SUM(read_count) as read_count
		 FROM reading_stats
		 WHERE user_id = ?
		 GROUP BY hour, weekday`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []HeatmapData
	for rows.Next() {
		var d HeatmapData
		if err := rows.Scan(&d.Hour, &d.Weekday, &d.ReadCount); err == nil {
			data = append(data, d)
		}
	}
	return data, nil
}
