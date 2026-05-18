package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

var client = &http.Client{
	Timeout: 30 * time.Second,
}

func DiscoverFeedURLs(siteURL string) ([]string, error) {
	if !strings.HasPrefix(siteURL, "http") {
		siteURL = "https://" + siteURL
	}

	parsedURL, err := url.Parse(siteURL)
	if err != nil {
		return nil, err
	}

	resp, err := client.Get(siteURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var feedURLs []string

	doc.Find(`link[type="application/rss+xml"], link[type="application/atom+xml"], link[type="application/rdf+xml"]`).Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if absoluteURL, err := parsedURL.Parse(href); err == nil {
				feedURLs = append(feedURLs, absoluteURL.String())
			}
		}
	})

	if len(feedURLs) == 0 {
		commonPaths := []string{
			"/feed", "/rss", "/atom", "/feed.xml", "/rss.xml", "/atom.xml",
			"/?feed=rss", "/?feed=atom", "/index.xml",
		}
		for _, path := range commonPaths {
			testURL := parsedURL.Scheme + "://" + parsedURL.Host + path
			if isValidFeed(testURL) {
				feedURLs = append(feedURLs, testURL)
				break
			}
		}
	}

	return feedURLs, nil
}

func isValidFeed(feedURL string) bool {
	resp, err := client.Get(feedURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	fp := gofeed.NewParser()
	_, err = fp.Parse(bytes.NewReader(body))
	return err == nil
}

func FetchFeed(feedURL string) (*gofeed.Feed, error) {
	fp := gofeed.NewParser()
	fp.Client = client

	var feed *gofeed.Feed
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}

		feed, err = fp.ParseURL(feedURL)
		if err == nil {
			return feed, nil
		}
	}

	return nil, fmt.Errorf("failed to fetch feed after 3 attempts: %w", err)
}

func ExtractFullContent(articleURL string) (string, error) {
	resp, err := client.Get(articleURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	doc.Find("script, style, nav, footer, header, aside").Remove()

	content := ""
	selectors := []string{
		"article", ".post-content", ".entry-content", ".article-content",
		"#content", ".content", "main", "[role='main']",
	}

	for _, selector := range selectors {
		el := doc.Find(selector).First()
		if el.Length() > 0 {
			html, err := el.Html()
			if err == nil && len(html) > 100 {
				content = html
				break
			}
		}
	}

	if content == "" {
		body := doc.Find("body")
		html, err := body.Html()
		if err != nil {
			return "", err
		}
		content = html
	}

	return content, nil
}
