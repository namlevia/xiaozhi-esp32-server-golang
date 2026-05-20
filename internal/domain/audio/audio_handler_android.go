//go:build android

package audio

import "errors"

type AudioProcesser struct {
	sampleRate       int
	channels         int
	perFrameDuration int
}

func GetAudioProcesser(sampleRate int, channels int, perFrameDuration int) (*AudioProcesser, error) {
	return &AudioProcesser{
		sampleRate:       sampleRate,
		channels:         channels,
		perFrameDuration: perFrameDuration,
	}, nil
}

func (a *AudioProcesser) Decoder(audio []byte, pcmData []int16) (int, error) {
	return 0, errors.New("Opus chưa hỗ trợ trên Android")
}

func (a *AudioProcesser) DecoderFloat32(audio []byte, pcmData []float32) (int, error) {
	return 0, errors.New("Opus chưa hỗ trợ trên Android")
}

func (a *AudioProcesser) Encoder(pcmData []int16, audio []byte) (int, error) {
	return 0, errors.New("Opus chưa hỗ trợ trên Android")
}
