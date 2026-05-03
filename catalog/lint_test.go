package catalog

import (
	"testing"
)

func TestIsJunkKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"bedrock/1024-x-1024/50-steps/bedrock/amazon.nova-canvas-v1:0", true},
		{"bedrock/512-x-512/max-steps/stability.stable-diffusion-xl-v0", true},
		{"bedrock/max-x-max/50-steps/stability.stable-diffusion-xl-v0", true},
		{"bedrock/1024-x-1024/50-steps/stability.stable-diffusion-xl-v1", true},
		{"bedrock/1024-x-1024/max-steps/stability.stable-diffusion-xl-v1", true},
		{"bedrock/512-x-512/50-steps/stability.stable-diffusion-xl-v0", true},
		{"bedrock/max-x-max/max-steps/stability.stable-diffusion-xl-v0", true},
		// Clean keys — should NOT be flagged.
		{"bedrock/anthropic.claude-sonnet-4-5-v1:0", false},
		{"openai/gpt-4o", false},
		{"fireworks/accounts/fireworks/models/llama-v3-8b", false},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got := IsJunkKey(tc.key)
			if got != tc.want {
				t.Errorf("IsJunkKey(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestLintProviders(t *testing.T) {
	t.Run("clean entry produces no issues", func(t *testing.T) {
		tmpDir := t.TempDir()

		writeTestModel(t, tmpDir, "openai", "gpt-4o.yaml", Entry{
			Provider:    "openai",
			ModelID:     "gpt-4o",
			DisplayName: "GPT-4o",
			Mode:        "chat",
			Lifecycle:   Lifecycle{Status: "ga"},
			Source:      "https://openrouter.ai/models/openai/gpt-4o",
			Tier:        "flagship",
		})

		issues, err := Lint(tmpDir)
		if err != nil {
			t.Fatalf("Lint() error: %v", err)
		}

		if len(issues) != 0 {
			for _, issue := range issues {
				t.Errorf("unexpected issue: %s: %s: %s", issue.Severity, issue.Key, issue.Message)
			}
		}
	})

	t.Run("junk key detected as error", func(t *testing.T) {
		tmpDir := t.TempDir()

		writeTestModel(t, tmpDir, "bedrock", "1024-x-1024__50-steps__stability.stable-diffusion-xl-v1.yaml", Entry{
			Provider:    "bedrock",
			ModelID:     "1024-x-1024/50-steps/stability.stable-diffusion-xl-v1",
			DisplayName: "1024-x-1024/50-steps/stability.stable-diffusion-xl-v1",
			Mode:        "image",
			Lifecycle:   Lifecycle{Status: "ga"},
			Tier:        "standard",
		})

		issues, err := Lint(tmpDir)
		if err != nil {
			t.Fatalf("Lint() error: %v", err)
		}

		var errors []LintIssue
		for _, issue := range issues {
			if issue.Severity == "error" {
				errors = append(errors, issue)
			}
		}

		if len(errors) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errors))
		}

		if errors[0].Key != "bedrock/1024-x-1024/50-steps/stability.stable-diffusion-xl-v1" {
			t.Errorf("expected junk key, got %q", errors[0].Key)
		}
	})

	t.Run("duplicate model_id across providers detected as warning", func(t *testing.T) {
		tmpDir := t.TempDir()

		writeTestModel(t, tmpDir, "openai", "gpt-4o.yaml", Entry{
			Provider:    "openai",
			ModelID:     "gpt-4o",
			DisplayName: "GPT-4o",
			Mode:        "chat",
			Lifecycle:   Lifecycle{Status: "ga"},
			Source:      "https://openrouter.ai/models/openai/gpt-4o",
			Tier:        "flagship",
		})

		writeTestModel(t, tmpDir, "azure", "gpt-4o.yaml", Entry{
			Provider:    "azure",
			ModelID:     "gpt-4o",
			DisplayName: "GPT-4o (Azure)",
			Mode:        "chat",
			Lifecycle:   Lifecycle{Status: "ga"},
			Source:      "https://openrouter.ai/models/azure/gpt-4o",
			Tier:        "flagship",
		})

		issues, err := Lint(tmpDir)
		if err != nil {
			t.Fatalf("Lint() error: %v", err)
		}

		var warnings []LintIssue
		for _, issue := range issues {
			if issue.Severity == "warning" {
				warnings = append(warnings, issue)
			}
		}

		if len(warnings) != 2 {
			t.Fatalf("expected 2 warnings (one per file), got %d", len(warnings))
		}

		for _, w := range warnings {
			if w.Key != "openai/gpt-4o" && w.Key != "azure/gpt-4o" {
				t.Errorf("unexpected warning key: %q", w.Key)
			}
		}
	})
}
