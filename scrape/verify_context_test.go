package scrape

import (
	"testing"

	"github.com/ferro-labs/model-catalog/catalog"
)

func ip(v int) *int { return &v }

func ctxEntry(contextWindow, maxOutput int) catalog.Entry {
	return catalog.Entry{
		Provider: "openai", ModelID: "gpt-x",
		ContextWindow: contextWindow, MaxOutputTokens: maxOutput,
	}
}

func TestVerifyContext(t *testing.T) {
	entries := map[string]catalog.Entry{"openai/gpt-x": ctxEntry(128000, 16000)}

	t.Run("single agreeing source is medium", func(t *testing.T) {
		obs := []Observation{{Source: "openrouter", SourceURL: "u", Provider: "openai", ModelID: "gpt-x", ContextWindow: ip(128000), MaxOutput: ip(16000)}}
		proofs := VerifyContext(entries, obs)
		if len(proofs) != 1 || proofs[0].Confidence != ConfidenceMedium {
			t.Fatalf("got %+v, want one medium proof", proofs)
		}
	})

	t.Run("two agreeing sources is high", func(t *testing.T) {
		obs := []Observation{
			{Source: "openrouter", SourceURL: "u", Provider: "openai", ModelID: "gpt-x", ContextWindow: ip(128000)},
			{Source: "models_dev", SourceURL: "u", Provider: "openai", ModelID: "gpt-x", ContextWindow: ip(128000)},
		}
		proofs := VerifyContext(entries, obs)
		if len(proofs) != 1 || proofs[0].Confidence != ConfidenceHigh {
			t.Fatalf("got %+v, want one high proof", proofs)
		}
	})

	t.Run("disagreeing context yields no proof", func(t *testing.T) {
		obs := []Observation{{Source: "openrouter", Provider: "openai", ModelID: "gpt-x", ContextWindow: ip(999)}}
		if proofs := VerifyContext(entries, obs); len(proofs) != 0 {
			t.Fatalf("got %+v, want no proof on context mismatch", proofs)
		}
	})

	t.Run("observation with no context fields yields no proof", func(t *testing.T) {
		obs := []Observation{{Source: "litellm", Provider: "openai", ModelID: "gpt-x", InputPerM: f(3.0)}}
		if proofs := VerifyContext(entries, obs); len(proofs) != 0 {
			t.Fatalf("got %+v, want no proof when no context observed", proofs)
		}
	})
}
