package scrape

import "time"

// Confidence indicates how trustworthy a scraped observation is.
type Confidence string

const (
	ConfidenceHigh     Confidence = "high"     // >=2 sources agree
	ConfidenceMedium   Confidence = "medium"   // 1 source only
	ConfidenceConflict Confidence = "conflict" // sources disagree
)

// Observation is a single data point scraped from an external source.
type Observation struct {
	Source        string
	SourceURL     string
	ScrapedAt     time.Time
	Provider      string
	ModelID       string
	InputPerM     *float64
	OutputPerM    *float64
	CacheReadPerM *float64
	ContextWindow *int
	MaxOutput     *int
}

// Scraper fetches model metadata from an external source.
type Scraper interface {
	Name() string
	Scrape() ([]Observation, error)
}
