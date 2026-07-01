package scrape

import (
	"strings"
	"testing"
)

func TestFormatWeeklyReviewIsDeterministicAndSorted(t *testing.T) {
	// Intentionally unsorted, mixed-confidence diffs.
	result := ReconcileResult{
		Diffs: []Delta{
			{CatalogKey: "openai/gpt-4o", Field: "pricing.output_per_m_tokens", Current: "10", Scraped: "12", Confidence: ConfidenceMedium, Sources: []string{"models_dev"}},
			{CatalogKey: "anthropic/claude-x", Field: "pricing.input_per_m_tokens", Current: "3", Scraped: "5", Confidence: ConfidenceConflict, Sources: []string{"openrouter", "litellm"}},
			{CatalogKey: "openai/gpt-4o", Field: "pricing.input_per_m_tokens", Current: "2.5", Scraped: "3", Confidence: ConfidenceMedium, Sources: []string{"litellm"}},
			// High-confidence diffs are auto-applied and must NOT appear here.
			{CatalogKey: "openai/o1", Field: "pricing.input_per_m_tokens", Current: "1", Scraped: "2", Confidence: ConfidenceHigh, Sources: []string{"litellm", "models_dev"}},
		},
	}

	out1 := FormatWeeklyReview(result)
	out2 := FormatWeeklyReview(result)
	if out1 != out2 {
		t.Fatal("FormatWeeklyReview is not deterministic")
	}

	// High-confidence (auto-applied) diffs are excluded.
	if strings.Contains(out1, "openai/o1") {
		t.Errorf("high-confidence diff leaked into review:\n%s", out1)
	}

	// Single-source diffs for the same key are sorted by field
	// (input before output).
	iInput := strings.Index(out1, "gpt-4o` `pricing.input_per_m_tokens")
	iOutput := strings.Index(out1, "gpt-4o` `pricing.output_per_m_tokens")
	if iInput < 0 || iOutput < 0 || iInput > iOutput {
		t.Errorf("diffs not sorted by field:\n%s", out1)
	}

	// Conflict section is present.
	if !strings.Contains(out1, "anthropic/claude-x") {
		t.Errorf("conflict diff missing:\n%s", out1)
	}
}

func TestFormatWeeklyReviewEmptyIsStable(t *testing.T) {
	empty := FormatWeeklyReview(ReconcileResult{})
	if empty != FormatWeeklyReview(ReconcileResult{}) {
		t.Fatal("empty review not stable")
	}
	if strings.TrimSpace(empty) == "" {
		t.Fatal("empty review should still render a stable document, not an empty string")
	}
}
