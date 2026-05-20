//go:build !cgo || android

package ten_vad

import (
	"errors"
	"sync"
	"unsafe"
)

type TenVADDLL struct{}

var (
	globalTenVAD *TenVADDLL
	dllOnce      sync.Once
)

func GetInstance() *TenVADDLL {
	dllOnce.Do(func() {
		globalTenVAD = &TenVADDLL{}
	})
	return globalTenVAD
}

func (t *TenVADDLL) CreateInstance(hopSize int, threshold float32) (unsafe.Pointer, error) {
	return nil, errors.New("TEN-VAD chưa hỗ trợ trên nền tảng này")
}

func (t *TenVADDLL) ProcessAudio(handle unsafe.Pointer, audioData []int16) (float32, int32, error) {
	return 0, 0, errors.New("TEN-VAD chưa hỗ trợ trên nền tảng này")
}

func (t *TenVADDLL) DestroyInstance(handle unsafe.Pointer) error {
	return nil
}

func (t *TenVADDLL) GetVersion() string {
	return "unsupported"
}
