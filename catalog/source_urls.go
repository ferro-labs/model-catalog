package catalog

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProviderSourceURLs maps provider IDs to their canonical pricing/models page.
var ProviderSourceURLs = map[string]string{
	"ai21":              "https://www.ai21.com/pricing",
	"aiml":              "https://aimlapi.com/pricing",
	"aleph_alpha":       "https://www.aleph-alpha.com/pricing",
	"amazon_nova":       "https://aws.amazon.com/ai/generative-ai/nova/pricing/",
	"anthropic":         "https://docs.anthropic.com/en/docs/about-claude/models",
	"anyscale":          "https://www.anyscale.com/pricing",
	"assemblyai":        "https://www.assemblyai.com/pricing",
	"azure":             "https://azure.microsoft.com/en-us/pricing/details/cognitive-services/openai-service/",
	"azure_foundry":     "https://azure.microsoft.com/en-us/pricing/details/cognitive-services/openai-service/",
	"bedrock":           "https://aws.amazon.com/bedrock/pricing/",
	"cerebras":          "https://cerebras.ai/pricing",
	"cloudflare":        "https://developers.cloudflare.com/workers-ai/models/",
	"cohere":            "https://cohere.com/pricing",
	"deepinfra":         "https://deepinfra.com/pricing",
	"deepseek":          "https://api-docs.deepseek.com/quick_start/pricing",
	"fal_ai":            "https://fal.ai/pricing",
	"featherless_ai":    "https://featherless.ai/pricing",
	"fireworks":         "https://fireworks.ai/pricing",
	"friendliai":        "https://friendli.ai/pricing",
	"gigachat":          "https://developers.sber.ru/docs/ru/gigachat/models",
	"github_copilot":    "https://docs.github.com/en/copilot/about-github-copilot/plans-and-pricing-for-github-copilot",
	"gmi":               "https://cloud.gmi.ai/pricing",
	"gradient_ai":       "https://gradient.ai/pricing",
	"groq":              "https://groq.com/pricing/",
	"heroku":            "https://www.heroku.com/pricing",
	"hugging_face":      "https://huggingface.co/pricing",
	"hyperbolic":        "https://www.hyperbolic.xyz/pricing",
	"lambda_ai":         "https://lambda.ai/pricing",
	"lemonade":          "https://lemonade.social/models",
	"llamagate":         "https://llamagate.ai/pricing",
	"minimax":           "https://www.minimax.io/platform/pricing",
	"mistral":           "https://mistral.ai/technology/",
	"nlp_cloud":         "https://nlpcloud.com/pricing.html",
	"novita":            "https://novita.ai/pricing",
	"nvidia_nim":        "https://build.nvidia.com/nim",
	"ollama":            "https://ollama.com/library",
	"openai":            "https://openai.com/api/pricing/",
	"openrouter":        "https://openrouter.ai/models",
	"perplexity":        "https://docs.perplexity.ai/guides/pricing",
	"qwen":              "https://help.aliyun.com/zh/model-studio/getting-started/models",
	"replicate":         "https://replicate.com/pricing",
	"sagemaker":         "https://aws.amazon.com/sagemaker/pricing/",
	"sarvam":            "https://www.sarvam.ai/pricing",
	"snowflake":         "https://www.snowflake.com/en/data-cloud/cortex/pricing/",
	"stability":         "https://platform.stability.ai/pricing",
	"together":          "https://www.together.ai/pricing",
	"vercel_ai_gateway": "https://vercel.com/docs/ai-gateway#supported-providers",
	"vertex_ai":         "https://cloud.google.com/vertex-ai/generative-ai/pricing",
	"vertex_ai-ai21_models": "https://cloud.google.com/vertex-ai/generative-ai/pricing",
	"volcengine":            "https://www.volcengine.com/pricing?product=ark",
	"voyage":                "https://docs.voyageai.com/docs/pricing",
	"wandb":                 "https://wandb.ai/pricing",
	"watsonx":               "https://www.ibm.com/products/watsonx-ai/foundation-models",
}

// BackfillSourceFromProviderURLs fills empty source fields using static
// provider-level URLs. This is a bulk operation for providers where every
// model shares the same canonical pricing/docs page.
func BackfillSourceFromProviderURLs(providersDir string, dryRun bool) (int, error) {
	sourceMap := make(map[string]string)

	matches, err := filepath.Glob(filepath.Join(providersDir, "*/models/*.yaml"))
	if err != nil {
		return 0, fmt.Errorf("glob: %w", err)
	}

	for _, path := range matches {
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			continue
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			continue
		}

		if entry.Source != "" {
			continue
		}

		url, ok := ProviderSourceURLs[entry.Provider]
		if !ok {
			continue
		}

		key := entry.Provider + "/" + entry.ModelID
		sourceMap[key] = url
	}

	return BackfillSource(providersDir, sourceMap, dryRun)
}
