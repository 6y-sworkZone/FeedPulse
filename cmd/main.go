package main

import (
	"log"

	"feedpulse/internal/database"
	"feedpulse/internal/handlers"
	"feedpulse/internal/middleware"
	"feedpulse/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	c := cron.New()
	c.AddFunc("@every 15m", func() {
		services.CrawlAllFeeds()
	})
	c.AddFunc("@hourly", func() {
		log.Println("Processing hourly webhook digest...")
		services.ProcessHourlyWebhookDigest()
	})
	c.AddFunc("@daily", func() {
		log.Println("Processing daily webhook digest...")
		services.ProcessDailyWebhookDigest()
	})
	c.Start()
	defer c.Stop()

	r := gin.Default()

	r.Static("/static", "./web/static")
	r.LoadHTMLGlob("./web/templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	api := r.Group("/api")
	{
		api.POST("/register", handlers.Register)
		api.POST("/login", handlers.Login)

		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.GET("/user", handlers.GetCurrentUser)
			auth.GET("/settings", handlers.GetUserSettings)
			auth.PUT("/settings", handlers.UpdateUserSettings)

			auth.POST("/feeds", handlers.AddFeed)
			auth.GET("/feeds", handlers.GetFeeds)
			auth.DELETE("/feeds/:id", handlers.DeleteFeed)
			auth.GET("/feeds/discover", handlers.DiscoverFeeds)
			auth.POST("/feeds/import", handlers.ImportOPML)
			auth.GET("/feeds/export", handlers.ExportOPML)

			auth.POST("/groups", handlers.CreateGroup)
			auth.GET("/groups", handlers.GetGroups)
			auth.PUT("/groups/:id", handlers.UpdateGroup)
			auth.DELETE("/groups/:id", handlers.DeleteGroup)
			auth.POST("/groups/:group_id/feeds/:feed_id", handlers.AddFeedToGroup)
			auth.DELETE("/groups/:group_id/feeds/:feed_id", handlers.RemoveFeedFromGroup)

			auth.PUT("/feeds/:id/fetch-interval", handlers.UpdateFeedFetchInterval)

			auth.GET("/articles", handlers.GetArticles)
			auth.GET("/articles/:id", handlers.GetArticle)
			auth.PUT("/articles/:id/read", handlers.MarkArticleRead)
			auth.PUT("/articles/:id/star", handlers.StarArticle)
			auth.PUT("/articles/:id/later", handlers.MarkArticleLater)
			auth.GET("/articles/search", handlers.SearchArticles)

			auth.GET("/stats", handlers.GetStats)
			auth.GET("/stats/daily", handlers.GetDailyStats)
			auth.GET("/stats/feeds", handlers.GetFeedStats)
			auth.GET("/stats/heatmap", handlers.GetHeatmap)

			auth.GET("/webhooks", handlers.GetWebhooks)
			auth.POST("/webhooks", handlers.CreateWebhook)
			auth.DELETE("/webhooks/:id", handlers.DeleteWebhook)
		}

		api.GET("/feeds/popular", handlers.GetPopularFeeds)
	}

	log.Println("Starting FeedPulse server on :8765...")
	if err := r.Run(":8765"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
