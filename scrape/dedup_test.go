package scrape

import "testing"

func f(v float64) *float64 { return &v }

func TestDedupObservationsRemovesSameSourceDuplicates(t *testing.T) {
	// Same provider/model/source appearing twice (e.g. LiteLLM "deepseek-chat"
	// and "deepseek/deepseek-chat" both mapping to deepseek/deepseek-chat).
	obs := []Observation{
		{Source: "litellm", Provider: "deepseek", ModelID: "deepseek-chat", InputPerM: f(0.28)},
		{Source: "litellm", Provider: "deepseek", ModelID: "deepseek-chat", InputPerM: f(0.28)},
		{Source: "models_dev", Provider: "deepseek", ModelID: "deepseek-chat", InputPerM: f(0.27)},
	}
	got := DedupObservations(obs)
	if len(got) != 2 {
		t.Fatalf("got %d observations, want 2 (one per source)", len(got))
	}
	perSource := map[string]int{}
	for _, o := range got {
		perSource[o.Source]++
	}
	if perSource["litellm"] != 1 || perSource["models_dev"] != 1 {
		t.Fatalf("per-source counts = %v, want litellm:1 models_dev:1", perSource)
	}
}

func TestDedupObservationsIsOrderIndependent(t *testing.T) {
	// Two same-source observations that DISAGREE must resolve to the same
	// survivor regardless of input order (deterministic, no churn).
	a := Observation{Source: "litellm", Provider: "gemini", ModelID: "gemini-exp-1206", InputPerM: f(0)}
	b := Observation{Source: "litellm", Provider: "gemini", ModelID: "gemini-exp-1206", InputPerM: f(0.3)}

	fwd := DedupObservations([]Observation{a, b})
	rev := DedupObservations([]Observation{b, a})

	if len(fwd) != 1 || len(rev) != 1 {
		t.Fatalf("expected 1 survivor, got fwd=%d rev=%d", len(fwd), len(rev))
	}
	if *fwd[0].InputPerM != *rev[0].InputPerM {
		t.Fatalf("dedup not order-independent: fwd=%v rev=%v", *fwd[0].InputPerM, *rev[0].InputPerM)
	}
}

func TestDedupObservationsKeepsDistinctModels(t *testing.T) {
	obs := []Observation{
		{Source: "litellm", Provider: "openai", ModelID: "gpt-4o"},
		{Source: "litellm", Provider: "openai", ModelID: "gpt-4o-mini"},
		{Source: "openrouter", Provider: "openai", ModelID: "gpt-4o"},
	}
	if got := DedupObservations(obs); len(got) != 3 {
		t.Fatalf("distinct models collapsed: got %d, want 3", len(got))
	}
}
