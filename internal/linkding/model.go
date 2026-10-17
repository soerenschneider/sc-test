package linkding

import (
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

func BookmarksAsTable(data []Bookmark) ([]string, [][]string) {
	headers := []string{
		"Title",
		"Url",
		"Tags",
		"Added",
		"Modified",
	}

	ret := make([][]string, 0, len(data))
	for _, bookmark := range data {
		ret = append(ret, []string{bookmark.Title, bookmark.URL, strings.Join(bookmark.TagNames, ", "), humanize.Time(bookmark.DateAdded), humanize.Time(bookmark.DateModified)})
	}

	return headers, ret
}

type Bookmark struct {
	ID                    int64     `json:"id"`
	URL                   string    `json:"url"`
	Title                 string    `json:"title"`
	Description           string    `json:"description"`
	Notes                 string    `json:"notes"`
	WebArchiveSnapshotURL string    `json:"web_archive_snapshot_url"`
	FaviconURL            string    `json:"favicon_url"`
	PreviewImageURL       string    `json:"preview_image_url"`
	IsArchived            bool      `json:"is_archived"`
	Unread                bool      `json:"unread"`
	Shared                bool      `json:"shared"`
	TagNames              []string  `json:"tag_names"`
	DateAdded             time.Time `json:"date_added"`
	DateModified          time.Time `json:"date_modified"`
	WebsiteTitle          string    `json:"website_title"`
	WebsiteDescription    string    `json:"website_description"`
}

type BookmarksResponse struct {
	Results []Bookmark `json:"results"`
	Next    string     `json:"next"` // for pagination
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TagsResponse struct {
	Results []Tag  `json:"results"`
	Next    string `json:"next"`
}
