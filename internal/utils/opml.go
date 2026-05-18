package utils

import (
	"encoding/xml"
	"io"
	"time"
)

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

type Head struct {
	Title        string `xml:"title"`
	DateCreated  string `xml:"dateCreated,omitempty"`
	DateModified string `xml:"dateModified,omitempty"`
	OwnerName    string `xml:"ownerName,omitempty"`
	OwnerEmail   string `xml:"ownerEmail,omitempty"`
}

type Body struct {
	Outlines []Outline `xml:"outline"`
}

type Outline struct {
	Text     string    `xml:"text,attr"`
	Title    string    `xml:"title,attr,omitempty"`
	Type     string    `xml:"type,attr,omitempty"`
	XMLURL   string    `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string    `xml:"htmlUrl,attr,omitempty"`
	Outlines []Outline `xml:"outline,omitempty"`
}

func ParseOPML(r io.Reader) (*OPML, error) {
	var opml OPML
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&opml); err != nil {
		return nil, err
	}
	return &opml, nil
}

func GenerateOPML(feeds []struct{ Title, XMLURL, HTMLURL string }) ([]byte, error) {
	opml := OPML{
		Version: "2.0",
		Head: Head{
			Title:        "FeedPulse Subscriptions",
			DateCreated:  time.Now().Format(time.RFC1123),
			DateModified: time.Now().Format(time.RFC1123),
		},
		Body: Body{
			Outlines: make([]Outline, len(feeds)),
		},
	}

	for i, feed := range feeds {
		opml.Body.Outlines[i] = Outline{
			Text:    feed.Title,
			Title:   feed.Title,
			Type:    "rss",
			XMLURL:  feed.XMLURL,
			HTMLURL: feed.HTMLURL,
		}
	}

	return xml.MarshalIndent(opml, "", "  ")
}

func ExtractFeedURLs(opml *OPML) []struct{ Title, URL string } {
	var feeds []struct{ Title, URL string }
	var extract func(outlines []Outline)
	extract = func(outlines []Outline) {
		for _, outline := range outlines {
			if outline.XMLURL != "" {
				title := outline.Title
				if title == "" {
					title = outline.Text
				}
				feeds = append(feeds, struct {
					Title string
					URL   string
				}{Title: title, URL: outline.XMLURL})
			}
			if len(outline.Outlines) > 0 {
				extract(outline.Outlines)
			}
		}
	}
	extract(opml.Body.Outlines)
	return feeds
}
