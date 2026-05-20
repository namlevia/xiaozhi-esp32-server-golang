//go:build android

package util

import (
	"context"
	"errors"
	"io"

	"github.com/gopxl/beep"
)

type AudioDecoder struct {
	AudioFormat       string
	TargetAudioFormat string
}

func WavToOpus(wavData []byte, sampleRate int, channels int, bitRate int) ([][]byte, error) {
	return nil, errors.New("Opus chưa hỗ trợ trên Android")
}

func CreateAudioDecoder(ctx context.Context, pipeReader io.ReadCloser, outputOpusChan chan []byte, perFrameDurationMs int, AudioFormat string) (*AudioDecoder, error) {
	return &AudioDecoder{AudioFormat: AudioFormat, TargetAudioFormat: "opus"}, nil
}

func CreateAudioDecoderWithSampleRate(ctx context.Context, pipeReader io.ReadCloser, outputOpusChan chan []byte, perFrameDurationMs int, AudioFormat string, targetSampleRate int) (*AudioDecoder, error) {
	return &AudioDecoder{AudioFormat: AudioFormat, TargetAudioFormat: "opus"}, nil
}

func (d *AudioDecoder) WithFormat(format beep.Format) *AudioDecoder {
	return d
}

func (d *AudioDecoder) WithTargetAudioFormat(targetAudioFormat string) *AudioDecoder {
	d.TargetAudioFormat = targetAudioFormat
	return d
}

func (d *AudioDecoder) Run(startTs int64) error {
	return errors.New("Opus chưa hỗ trợ trên Android")
}

func WriteLengthPrefixedFrame(writer io.Writer, frame []byte) error {
	return errors.New("Opus chưa hỗ trợ trên Android")
}

func readLengthPrefixedFrame(reader io.Reader) ([]byte, error) {
	return nil, errors.New("Opus chưa hỗ trợ trên Android")
}

func NormalizeOpusSampleRate(sampleRate int) int {
	if sampleRate <= 0 {
		return 16000
	}
	return sampleRate
}

func PCM16ToOggOpus(samples []int16, sampleRate int, channels int, frameDurationMs int) ([]byte, error) {
	return nil, errors.New("Opus chưa hỗ trợ trên Android")
}

func WrapOggOpusPackets(packets [][]byte, sampleRate int, channels int, frameSizePerChannel int) []byte {
	return nil
}

func GetAudioFormatByMimeType(mimeType string) string {
	switch mimeType {
	case "audio/mpeg", "audio/mp3", "audio/mpeg3", "audio/x-mpeg-3":
		return "mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return "wav"
	case "audio/pcm", "audio/x-pcm":
		return "pcm"
	case "audio/ogg", "application/ogg":
		return "ogg_opus"
	case "audio/opus":
		return "opus"
	default:
		return "mp3"
	}
}
