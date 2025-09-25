package speex

/*
#cgo pkg-config: speexdsp
#include <speex/speex_preprocess.h>
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"

	"open_tool_speex/pkg/types"
)

// Preprocessor wraps standalone Speex Preprocessor (without echo state)
type Preprocessor struct {
	preprocState *C.SpeexPreprocessState
	frameSize    int
}

// NewPreprocessor creates new standalone Speex Preprocessor with default settings
func NewPreprocessor(frameSize, sampleRate int) (*Preprocessor, error) {
	config := types.NSConfig{
		NoiseSuppress: -15.0,
		EnableVAD:     false,
		VADProbStart:  80,
		VADProbCont:   65,
		EnableAGC:     false,
		AGCLevel:      30000.0,
	}
	return NewPreprocessorWithConfig(frameSize, sampleRate, config)
}

// NewPreprocessorWithConfig creates new standalone Speex Preprocessor with custom configuration
func NewPreprocessorWithConfig(frameSize, sampleRate int, config types.NSConfig) (*Preprocessor, error) {
	if frameSize <= 0 || sampleRate <= 0 {
		return nil, errors.New("invalid parameters")
	}

	preprocState := C.speex_preprocess_state_init(C.int(frameSize), C.int(sampleRate))
	if preprocState == nil {
		return nil, errors.New("failed to create preprocessor state")
	}

	// Configure denoising
	val := C.int(1)
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_DENOISE, unsafe.Pointer(&val))

	// Configure noise suppression level
	noiseLevel := C.int(int(config.NoiseSuppress))
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_NOISE_SUPPRESS, unsafe.Pointer(&noiseLevel))

	// Configure VAD if enabled
	if config.EnableVAD {
		vadVal := C.int(1)
		C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_VAD, unsafe.Pointer(&vadVal))

		probStart := C.int(config.VADProbStart)
		C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_PROB_START, unsafe.Pointer(&probStart))

		probCont := C.int(config.VADProbCont)
		C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_PROB_CONTINUE, unsafe.Pointer(&probCont))
	}

	// Configure AGC if enabled
	if config.EnableAGC {
		agcVal := C.int(1)
		C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_AGC, unsafe.Pointer(&agcVal))

		agcLevel := C.int(int(config.AGCLevel))
		C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_AGC_LEVEL, unsafe.Pointer(&agcLevel))
	}

	return &Preprocessor{
		preprocState: preprocState,
		frameSize:    frameSize,
	}, nil
}

// ProcessFrame processes a frame with noise suppression
func (ns *Preprocessor) ProcessFrame(inputFrame []int16) []int16 {
	if len(inputFrame) != ns.frameSize {
		return nil
	}

	output := make([]int16, ns.frameSize)
	// Copy input to output first
	copy(output, inputFrame)
	outPtr := (*C.spx_int16_t)(unsafe.Pointer(&output[0]))

	// Apply preprocessing (noise suppression, VAD, AGC)
	C.speex_preprocess_run(ns.preprocState, outPtr)

	return output
}

// Destroy cleans up resources
func (ns *Preprocessor) Destroy() {
	if ns.preprocState != nil {
		C.speex_preprocess_state_destroy(ns.preprocState)
	}
}
