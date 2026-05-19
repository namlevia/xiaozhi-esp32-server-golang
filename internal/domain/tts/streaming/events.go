package streaming

// SentenceSignalType noi_dungaudiotrướcnoi_dunggửinoi_dungloại。
type SentenceSignalType string

const (
	SentenceSignalStart SentenceSignalType = "sentence_start"
	SentenceSignalEnd   SentenceSignalType = "sentence_end"
)

// SentenceSignal noi_dunghiện tạiaudionoi_dungvừanoi_dung。
type SentenceSignal struct {
	Type SentenceSignalType
	Text string
}

// SynthesisEvent noi_dungdualstreaming TTS output。
// Audio làhiện tạiaudionoi_dung；SentenceSignals noi_dunggửinoi_dungaudionoi_dungtrướcnoi_dunggửinoi_dungvừanoi_dung。
type SynthesisEvent struct {
	Audio           []byte
	SentenceSignals []SentenceSignal
}
