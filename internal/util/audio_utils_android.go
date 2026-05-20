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
