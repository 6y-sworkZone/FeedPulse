package services

import (
	"strings"
	"time"

	"feedpulse/internal/database"
	"feedpulse/internal/models"
)

func GetArticles(userID int64, feedID *int64, groupID *int64, isRead *bool, isStarred *bool, isLater *bool, page, perPage int) ([]models.Article, int, error) {
	query := `
		SELECT a.id, a.feed_id, a.user_id, a.guid, a.title, a.url, a.summary,
		       a.content, a.author, a.published_at, a.is_read, a.is_starred,
		       a.is_later, a.read_at, a.created_at
		FROM articles a
		WHERE a.user_id = ?
	`
	args := []interface{}{userID}

	if feedID != nil {
		query += " AND a.feed_id = ?"
		args = append(args, *feedID)
	}

	if groupID != nil {
		query += ` AND a.feed_id IN (SELECT feed_id FROM feed_groups WHERE group_id = ?)`
		args = append(args, *groupID)
	}

	if isRead != nil {
		query += " AND a.is_read = ?"
		args = append(args, *isRead)
	}

	if isStarred != nil && *isStarred {
		query += " AND a.is_starred = 1"
	}

	if isLater != nil && *isLater {
		query += " AND a.is_later = 1"
	}

	countQuery := query
	countQuery = strings.Replace(countQuery, "SELECT a.id", "SELECT COUNT(*)", 1)

	var total int
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += " ORDER BY a.published_at DESC LIMIT ? OFFSET ?"
	args = append(args, perPage, (page-1)*perPage)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.UserID, &article.GUID,
			&article.Title, &article.URL, &article.Summary, &article.Content,
			&article.Author, &article.PublishedAt, &article.IsRead, &article.IsStarred,
			&article.IsLater, &article.ReadAt, &article.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		articles = append(articles, article)
	}

	return articles, total, nil
}

func GetArticleByID(userID, articleID int64) (*models.Article, error) {
	var article models.Article
	err := database.DB.QueryRow(
		`SELECT id, feed_id, user_id, guid, title, url, summary, content, author,
		        published_at, is_read, is_starred, is_later, read_at, created_at
		 FROM articles WHERE id = ? AND user_id = ?`,
		articleID, userID,
	).Scan(&article.ID, &article.FeedID, &article.UserID, &article.GUID, &article.Title,
		&article.URL, &article.Summary, &article.Content, &article.Author,
		&article.PublishedAt, &article.IsRead, &article.IsStarred, &article.IsLater,
		&article.ReadAt, &article.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func MarkArticleRead(userID, articleID int64, read bool) error {
	now := time.Now()
	var readAt *time.Time
	if read {
		readAt = &now
	}

	_, err := database.DB.Exec(
		"UPDATE articles SET is_read = ?, read_at = ? WHERE id = ? AND user_id = ?",
		read, readAt, articleID, userID,
	)

	if err == nil && read {
		UpdateReadingStats(userID, now)
	}

	return err
}

func StarArticle(userID, articleID int64, starred bool) error {
	_, err := database.DB.Exec(
		"UPDATE articles SET is_starred = ? WHERE id = ? AND user_id = ?",
		starred, articleID, userID,
	)
	return err
}

func MarkArticleLater(userID, articleID int64, later bool) error {
	_, err := database.DB.Exec(
		"UPDATE articles SET is_later = ? WHERE id = ? AND user_id = ?",
		later, articleID, userID,
	)
	return err
}

type SearchFilters struct {
	FeedID     *int64
	StartDate  *string
	EndDate    *string
}

var fts5Available = true

func checkFTS5Available() bool {
	var name string
	err := database.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='articles_fts'").Scan(&name)
	return err == nil && name != ""
}

func SearchArticles(userID int64, query string, filters SearchFilters, page, perPage int) ([]models.Article, int, error) {
	searchPattern := "%" + query + "%"
	
	if fts5Available && checkFTS5Available() {
		return searchWithFTS5(userID, query, filters, page, perPage)
	}
	
	return searchWithLike(userID, searchPattern, filters, page, perPage)
}

func buildFilterConditions(filters SearchFilters) (string, []interface{}) {
	conditions := ""
	args := []interface{}{}

	if filters.FeedID != nil {
		conditions += " AND a.feed_id = ?"
		args = append(args, *filters.FeedID)
	}

	if filters.StartDate != nil && *filters.StartDate != "" {
		conditions += " AND a.published_at >= ?"
		args = append(args, *filters.StartDate)
	}

	if filters.EndDate != nil && *filters.EndDate != "" {
		conditions += " AND a.published_at <= ?"
		args = append(args, *filters.EndDate)
	}

	return conditions, args
}

func searchWithFTS5(userID int64, query string, filters SearchFilters, page, perPage int) ([]models.Article, int, error) {
	filterConditions, filterArgs := buildFilterConditions(filters)
	
	countQuery := `
		SELECT COUNT(*)
		FROM articles_fts fts
		JOIN articles a ON fts.rowid = a.id
		WHERE a.user_id = ? AND articles_fts MATCH ?
	` + filterConditions

	args := append([]interface{}{userID, query}, filterArgs...)

	var total int
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		fts5Available = false
		return searchWithLike(userID, "%"+query+"%", filters, page, perPage)
	}

	searchQuery := `
		SELECT a.id, a.feed_id, a.user_id, a.guid, a.title, a.url, a.summary,
		       a.content, a.author, a.published_at, a.is_read, a.is_starred,
		       a.is_later, a.read_at, a.created_at
		FROM articles_fts fts
		JOIN articles a ON fts.rowid = a.id
		WHERE a.user_id = ? AND articles_fts MATCH ?
	` + filterConditions + `
		ORDER BY rank, a.published_at DESC
		LIMIT ? OFFSET ?
	`

	args = append([]interface{}{userID, query}, filterArgs...)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := database.DB.Query(searchQuery, args...)
	if err != nil {
		fts5Available = false
		return searchWithLike(userID, "%"+query+"%", filters, page, perPage)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.UserID, &article.GUID,
			&article.Title, &article.URL, &article.Summary, &article.Content,
			&article.Author, &article.PublishedAt, &article.IsRead, &article.IsStarred,
			&article.IsLater, &article.ReadAt, &article.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		articles = append(articles, article)
	}

	return articles, total, nil
}

func searchWithLike(userID int64, pattern string, filters SearchFilters, page, perPage int) ([]models.Article, int, error) {
	filterConditions, filterArgs := buildFilterConditions(filters)
	
	countQuery := `
		SELECT COUNT(*)
		FROM articles a
		WHERE a.user_id = ? AND (a.title LIKE ? OR a.content LIKE ? OR a.author LIKE ?)
	` + filterConditions

	args := append([]interface{}{userID, pattern, pattern, pattern}, filterArgs...)

	var total int
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	searchQuery := `
		SELECT a.id, a.feed_id, a.user_id, a.guid, a.title, a.url, a.summary,
		       a.content, a.author, a.published_at, a.is_read, a.is_starred,
		       a.is_later, a.read_at, a.created_at
		FROM articles a
		WHERE a.user_id = ? AND (a.title LIKE ? OR a.content LIKE ? OR a.author LIKE ?)
	` + filterConditions + `
		ORDER BY a.published_at DESC
		LIMIT ? OFFSET ?
	`

	args = append([]interface{}{userID, pattern, pattern, pattern}, filterArgs...)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := database.DB.Query(searchQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		err := rows.Scan(&article.ID, &article.FeedID, &article.UserID, &article.GUID,
			&article.Title, &article.URL, &article.Summary, &article.Content,
			&article.Author, &article.PublishedAt, &article.IsRead, &article.IsStarred,
			&article.IsLater, &article.ReadAt, &article.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		articles = append(articles, article)
	}

	return articles, total, nil
}

func UpdateReadingStats(userID int64, t time.Time) error {
	date := t.Format("2006-01-02")
	hour := t.Hour()
	weekday := int(t.Weekday())

	_, err := database.DB.Exec(
		`INSERT INTO reading_stats (user_id, date, hour, weekday, read_count)
		 VALUES (?, ?, ?, ?, 1)
		 ON CONFLICT(user_id, date, hour) DO UPDATE SET read_count = read_count + 1`,
		userID, date, hour, weekday,
	)
	return err
}
