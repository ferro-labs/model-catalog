package catalog

// Model invocation modes. Mode describes the provider endpoint or request
// contract required to invoke an entry; it does not describe model
// capabilities, accepted media, pricing units, or lifecycle.
const (
	ModeChat       = "chat"
	ModeCompletion = "completion"
	ModeResponses  = "responses"
	ModeEmbedding  = "embedding"
	ModeImage      = "image"
	ModeAudioIn    = "audio_in"
	ModeAudioOut   = "audio_out"
	ModeVideo      = "video"
	ModeRealtime   = "realtime"
	ModeAgent      = "agent"
	ModeOCR        = "ocr"
	ModeRerank     = "rerank"
	ModeModeration = "moderation"
	ModeTool       = "tool"
)

var modeValues = []string{
	ModeChat,
	ModeCompletion,
	ModeResponses,
	ModeEmbedding,
	ModeImage,
	ModeAudioIn,
	ModeAudioOut,
	ModeVideo,
	ModeRealtime,
	ModeAgent,
	ModeOCR,
	ModeRerank,
	ModeModeration,
	ModeTool,
}

var validModes = func() map[string]bool {
	modes := make(map[string]bool, len(modeValues))
	for _, mode := range modeValues {
		modes[mode] = true
	}
	return modes
}()

// ModeValues returns the accepted mode values in their documented order.
func ModeValues() []string {
	return append([]string(nil), modeValues...)
}

// IsValidMode reports whether mode is part of the catalog schema.
func IsValidMode(mode string) bool {
	return validModes[mode]
}
