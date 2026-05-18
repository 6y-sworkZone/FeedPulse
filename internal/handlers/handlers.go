package handlers

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"

	"feedpulse/internal/services"
	"feedpulse/internal/utils"

	"github.com/gin-gonic/gin"
)

func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.RegisterUser(req.Username, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.LoginUser(req.Email, req.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

func GetCurrentUser(c *gin.Context) {
	userID := c.GetInt64("userID")

	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func GetUserSettings(c *gin.Context) {
	userID := c.GetInt64("userID")

	settings, err := services.GetUserSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

func UpdateUserSettings(c *gin.Context) {
	userID := c.GetInt64("userID")

	var req struct {
		Theme          string `json:"theme"`
		FontSize       string `json:"font_size"`
		ArticlesPerPage int   `json:"articles_per_page"`
		ReadingPrefs   string `json:"reading_preferences"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := services.UpdateUserSettings(userID, req.Theme, req.FontSize, req.ArticlesPerPage, req.ReadingPrefs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

func AddFeed(c *gin.Context) {
	userID := c.GetInt64("userID")

	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feed, err := services.AddFeed(userID, req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add feed"})
		return
	}

	c.JSON(http.StatusOK, feed)
}

func GetFeeds(c *gin.Context) {
	userID := c.GetInt64("userID")

	feeds, err := services.GetUserFeeds(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feeds"})
		return
	}

	c.JSON(http.StatusOK, feeds)
}

func DeleteFeed(c *gin.Context) {
	userID := c.GetInt64("userID")
	feedID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})
		return
	}

	if err := services.DeleteFeed(userID, feedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete feed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feed deleted"})
}

func DiscoverFeeds(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL required"})
		return
	}

	feeds, err := utils.DiscoverFeedURLs(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to discover feeds"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"feeds": feeds})
}

func CreateGroup(c *gin.Context) {
	userID := c.GetInt64("userID")

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := services.CreateGroup(userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func GetGroups(c *gin.Context) {
	userID := c.GetInt64("userID")

	groups, err := services.GetUserGroups(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get groups"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func AddFeedToGroup(c *gin.Context) {
	userID := c.GetInt64("userID")
	_ = userID
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}
	feedID, err := strconv.ParseInt(c.Param("feedId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})
		return
	}

	if err := services.AddFeedToGroup(feedID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add feed to group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feed added to group"})
}

func RemoveFeedFromGroup(c *gin.Context) {
	userID := c.GetInt64("userID")
	_ = userID
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}
	feedID, err := strconv.ParseInt(c.Param("feedId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feed ID"})
		return
	}

	if err := services.RemoveFeedFromGroup(feedID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove feed from group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feed removed from group"})
}

func DeleteGroup(c *gin.Context) {
	userID := c.GetInt64("userID")
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	if err := services.DeleteGroup(userID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted"})
}

func GetArticles(c *gin.Context) {
	userID := c.GetInt64("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	var feedID *int64
	if fidStr := c.Query("feed_id"); fidStr != "" {
		if fid, err := strconv.ParseInt(fidStr, 10, 64); err == nil {
			feedID = &fid
		}
	}

	var groupID *int64
	if gidStr := c.Query("group_id"); gidStr != "" {
		if gid, err := strconv.ParseInt(gidStr, 10, 64); err == nil {
			groupID = &gid
		}
	}

	var isRead *bool
	if readStr := c.Query("is_read"); readStr != "" {
		if read, err := strconv.ParseBool(readStr); err == nil {
			isRead = &read
		}
	}

	var isStarred *bool
	if starredStr := c.Query("is_starred"); starredStr == "true" {
		s := true
		isStarred = &s
	}

	var isLater *bool
	if laterStr := c.Query("is_later"); laterStr == "true" {
		l := true
		isLater = &l
	}

	articles, total, err := services.GetArticles(userID, feedID, groupID, isRead, isStarred, isLater, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get articles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"articles": articles, "total": total, "page": page, "per_page": perPage})
}

func GetArticle(c *gin.Context) {
	userID := c.GetInt64("userID")
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	article, err := services.GetArticleByID(userID, articleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	c.JSON(http.StatusOK, article)
}

func MarkArticleRead(c *gin.Context) {
	userID := c.GetInt64("userID")
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req struct {
		Read bool `json:"read"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Read = true
	}

	if err := services.MarkArticleRead(userID, articleID, req.Read); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated"})
}

func StarArticle(c *gin.Context) {
	userID := c.GetInt64("userID")
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req struct {
		Starred bool `json:"starred"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Starred = true
	}

	if err := services.StarArticle(userID, articleID, req.Starred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated"})
}

func MarkArticleLater(c *gin.Context) {
	userID := c.GetInt64("userID")
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req struct {
		Later bool `json:"later"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Later = true
	}

	if err := services.MarkArticleLater(userID, articleID, req.Later); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated"})
}

func SearchArticles(c *gin.Context) {
	userID := c.GetInt64("userID")
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	articles, total, err := services.SearchArticles(userID, query, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search articles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"articles": articles, "total": total, "page": page, "per_page": perPage})
}

func GetStats(c *gin.Context) {
	userID := c.GetInt64("userID")

	stats, err := services.GetUserStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func GetDailyStats(c *gin.Context) {
	userID := c.GetInt64("userID")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	stats, err := services.GetDailyReadingStats(userID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func GetFeedStats(c *gin.Context) {
	userID := c.GetInt64("userID")

	stats, err := services.GetFeedArticleStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func GetHeatmap(c *gin.Context) {
	userID := c.GetInt64("userID")

	data, err := services.GetReadingHeatmap(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get heatmap"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func GetPopularFeeds(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	feeds, err := services.GetPopularFeeds(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get popular feeds"})
		return
	}

	c.JSON(http.StatusOK, feeds)
}

func ImportOPML(c *gin.Context) {
	userID := c.GetInt64("userID")

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
		return
	}

	if err := services.ImportOPML(userID, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import OPML"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OPML imported successfully"})
}

func ExportOPML(c *gin.Context) {
	userID := c.GetInt64("userID")

	data, err := services.ExportOPML(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export OPML"})
		return
	}

	c.Header("Content-Type", "application/xml")
	c.Header("Content-Disposition", "attachment; filename=feeds.opml")
	c.Data(http.StatusOK, "application/xml", data)
}

func GetWebhooks(c *gin.Context) {
	userID := c.GetInt64("userID")

	configs, err := services.GetWebhookConfigs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get webhooks"})
		return
	}

	c.JSON(http.StatusOK, configs)
}

func CreateWebhook(c *gin.Context) {
	userID := c.GetInt64("userID")

	var req struct {
		URL       string `json:"url" binding:"required"`
		Enabled   bool   `json:"enabled"`
		Frequency string `json:"frequency"`
		Keywords  string `json:"keywords"`
		FeedIDs   string `json:"feed_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Frequency == "" {
		req.Frequency = "realtime"
	}

	config, err := services.CreateWebhookConfig(userID, req.URL, req.Enabled, req.Frequency, req.Keywords, req.FeedIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook"})
		return
	}

	c.JSON(http.StatusOK, config)
}

func DeleteWebhook(c *gin.Context) {
	userID := c.GetInt64("userID")
	configID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	if err := services.DeleteWebhookConfig(userID, configID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
}
