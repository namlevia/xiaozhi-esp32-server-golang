package edge

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/difyz9/edge-tts-go/pkg/communicate"
)

// EdgeTTSProvider Edge TTS provider
// hỗ trợnoi_dungvàstreamingTTS，outputOpusframe
// configtham số：voice, rate, volume, pitch, connectTimeout, receiveTimeout
type EdgeTTSProvider struct {
	Voice          string
	Rate           string
	Volume         string
	Pitch          string
	ConnectTimeout int
	ReceiveTimeout int
}

// NewEdgeTTSProvider tạoEdgeTTSProvider
func NewEdgeTTSProvider(config map[string]interface{}) *EdgeTTSProvider {
	voice, _ := config["voice"].(string)
	rate, _ := config["rate"].(string)
	volume, _ := config["volume"].(string)
	pitch, _ := config["pitch"].(string)
	connectTimeout, _ := config["connect_timeout"].(int)
	receiveTimeout, _ := config["receive_timeout"].(int)
	if rate == "" {
		rate = "+0%"
	}
	if volume == "" {
		volume = "+0%"
	}
	if pitch == "" {
		pitch = "+0Hz"
	}
	if connectTimeout == 0 {
		connectTimeout = 10
	}
	if receiveTimeout == 0 {
		receiveTimeout = 60
	}
	return &EdgeTTSProvider{
		Voice:          voice,
		Rate:           rate,
		Volume:         volume,
		Pitch:          pitch,
		ConnectTimeout: connectTimeout,
		ReceiveTimeout: receiveTimeout,
	}
}

// TextToSpeech noi_dungtổng hợp，trả vềOpusframe
func (p *EdgeTTSProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	startTs := time.Now().UnixMilli()
	// noi_dungMP3noi_dung
	tmpFile := fmt.Sprintf("/tmp/edge-tts-%d.mp3", time.Now().UnixNano())
	defer os.Remove(tmpFile)

	comm, err := communicate.NewCommunicate(
		text,
		p.Voice,
		p.Rate,
		p.Volume,
		p.Pitch,
		"", // proxy
		p.ConnectTimeout,
		p.ReceiveTimeout,
	)
	if err != nil {
		log.Errorf("EdgeTTS Communicatetạothất bại: %v", err)
		return nil, err
	}
	// noi_dungMP3
	err = comm.Save(ctx, tmpFile, "")
	if err != nil {
		log.Errorf("EdgeTTSnoi_dungMP3thất bại: %v", err)
		return nil, err
	}
	// MP3noi_dungOpus
	f, err := os.Open(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("noi_dungMP3thất bại: %v", err)
	}
	defer f.Close()
	pipeReader, pipeWriter := io.Pipe()
	outputChan := make(chan []byte, 1000)
	// ghiMP3datatớipipe
	go func() {
		_, _ = io.Copy(pipeWriter, f)
		pipeWriter.Close()
	}()
	mp3Decoder, err := util.CreateAudioDecoder(ctx, pipeReader, outputChan, frameDuration, "mp3")
	if err != nil {
		return nil, fmt.Errorf("tạoMP3decoderthất bại: %v", err)
	}
	var opusFrames [][]byte
	done := make(chan struct{})
	go func() {
		for frame := range outputChan {
			opusFrames = append(opusFrames, frame)
		}
		done <- struct{}{}
	}()
	if err := mp3Decoder.Run(startTs); err != nil {
		return nil, fmt.Errorf("MP3noi_dungthất bại: %v", err)
	}
	<-done
	return opusFrames, nil
}

// TextToSpeechStream streamingtổng hợp，trả vềOpusframechan
func (p *EdgeTTSProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (chan []byte, error) {
	startTs := time.Now().UnixMilli()
	comm, err := communicate.NewCommunicate(
		text,
		p.Voice,
		p.Rate,
		p.Volume,
		p.Pitch,
		"", // proxy
		p.ConnectTimeout,
		p.ReceiveTimeout,
	)
	if err != nil {
		log.Errorf("EdgeTTS Communicatetạothất bại: %v", err)
		return nil, err
	}

	chunkChan, errChan := comm.Stream(ctx)
	outputChan := make(chan []byte, 100)
	pipeReader, pipeWriter := io.Pipe()
	// MP3noi_dungOpusdecoder
	go func() {
		defer func() {
			pipeWriter.Close()
			log.Debugf("EdgeTTSstreamingtổng hợpnoi_dung, noi_dung: %d ms", time.Now().UnixMilli()-startTs)
			if err := <-errChan; err != nil {
				log.Errorf("EdgeTTSstreamingtổng hợpnoi_dung: %v", err)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				log.Debugf("EdgeTTS Stream context done, exit")
				return
			default:
				select {
				case chunk, ok := <-chunkChan:
					if !ok {
						log.Debugf("EdgeTTS Stream channel closed, exit")
						return
					}
					if chunk.Type == "audio" {
						_, _ = pipeWriter.Write(chunk.Data)
					}
				}
			}
		}

	}()
	// Khởi độngMP3→Opusnoi_dung
	go func() {
		mp3Decoder, err := util.CreateAudioDecoder(ctx, pipeReader, outputChan, frameDuration, "mp3")
		if err != nil {
			log.Errorf("EdgeTTS MP3decodertạothất bại: %v", err)
			return
		}
		if err := mp3Decoder.Run(startTs); err != nil {
			log.Errorf("EdgeTTS MP3noi_dungthất bại: %v", err)
		}
		log.Debugf("EdgeTTS MP3noi_dung, noi_dung: %d ms", time.Now().UnixMilli()-startTs)
	}()
	return outputChan, nil
}

// SetVoice Thiết lập tham số voice
func (p *EdgeTTSProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("noi_dungvoiceconfig: noi_dung voice")
}

// Close đóngtài nguyên（noi_dungtrạng thái Provider，noi_dungđóng）
func (p *EdgeTTSProvider) Close() error {
	return nil
}

// IsValid Kiểm tra tài nguyên có hợp lệ không
func (p *EdgeTTSProvider) IsValid() bool {
	return p != nil
}
