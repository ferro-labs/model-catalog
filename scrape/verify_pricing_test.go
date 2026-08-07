package scrape

import (
	"testing"

	"github.com/ferro-labs/model-catalog/catalog"
)

func entry(input, output float64) catalog.Entry {
	return catalog.Entry{
		Provider: "openai", ModelID: "gpt-x",
		Pricing: catalog.Pricing{
			InputPerMTokens:  catalog.NewNullFloat64(input),
			OutputPerMTokens: catalog.NewNullFloat64(output),
		},
	}
}

func TestVerifyPricing(t *testing.T) {
	entries := map[string]catalog.Entry{"openai/gpt-x": entry(3.0, 15.0)}

	t.Run("single agreeing source is medium", func(t *testing.T) {
		obs := []Observation{{Source: "openrouter", SourceURL: "u", Provider: "openai", ModelID: "gpt-x", InputPerM: f(3.0), OutputPerM: f(15.0)}}
		proofs := VerifyPricing(entries, obs)
		if len(proofs) != 1 || proofs[0].Confidence != ConfidenceMedium {
			t.Fatalf("got %+v, want one medium proof", proofs)
		}
	})

	t.Run("two agreeing sources is high", func(t *testing.T) {
		obs := []Observation{
			{Source: "openrouter", SourceURL: "u", Provider: "openai", ModelID: "gpt-x", InputPerM: f(3.0), OutputPerM: f(15.0)},
			{Source: "models_dev", SourceURL: "u", Provider: "openai", ModelID: "gpt-x", InputPerM: f(3.0), OutputPerM: f(15.0)},
		}
		proofs := VerifyPricing(entries, obs)
		if len(proofs) != 1 || proofs[0].Confidence != ConfidenceHigh {
			t.Fatalf("got %+v, want one high proof", proofs)
		}
		if len(proofs[0].Sources) != 2 {
			t.Errorf("want 2 sources, got %d", len(proofs[0].Sources))
		}
	})

	t.Run("disagreeing source yields no proof", func(t *testing.T) {
		obs := []Observation{{Source: "openrouter", SourceURL: "u", Provider: "openai", ModelID: "gpt-x", InputPerM: f(9.9), OutputPerM: f(15.0)}}
		if proofs := VerifyPricing(entries, obs); len(proofs) != 0 {
			t.Fatalf("got %+v, want no proof on price mismatch", proofs)
		}
	})

	t.Run("observation with no pricing yields no proof", func(t *testing.T) {
		obs := []Observation{{Source: "api", Provider: "openai", ModelID: "gpt-x"}}
		if proofs := VerifyPricing(entries, obs); len(proofs) != 0 {
			t.Fatalf("got %+v, want no proof when no price observed", proofs)
		}
	})

	t.Run("observation for unknown model is ignored", func(t *testing.T) {
		obs := []Observation{{Source: "openrouter", Provider: "openai", ModelID: "ghost", InputPerM: f(1.0)}}
		if proofs := VerifyPricing(entries, obs); len(proofs) != 0 {
			t.Fatalf("got %+v, want no proof for uncatalogued model", proofs)
		}
	})
}
