package speex

/*
#cgo pkg-config: speexdsp
#include <speex/speex_echo.h>
#include <speex/speex_preprocess.h>
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

// AEC wraps Speex Echo Canceller and Preprocessor
type AEC struct {
	echoState    *C.SpeexEchoState
	preprocState *C.SpeexPreprocessState
	frameSize    int
	filterLen    int
}

// NewAEC creates new Speex AEC instance
// frameSize: samples per frame (320 for 20ms at 16kHz)
// filterLen: echo tail length in samples (3200 for 200ms at 16kHz)
// sampleRate: sample rate in Hz (16000)
func NewAEC(frameSize, filterLen, sampleRate int) (*AEC, error) {
	if frameSize <= 0 || filterLen <= 0 || sampleRate <= 0 {
		return nil, errors.New("invalid parameters")
	}

	// Create echo state
	echoState := C.speex_echo_state_init(C.int(frameSize), C.int(filterLen))
	if echoState == nil {
		return nil, errors.New("failed to create echo state")
	}

	// Set sample rate
	rate := C.int(sampleRate)
	C.speex_echo_ctl(echoState, C.SPEEX_ECHO_SET_SAMPLING_RATE, unsafe.Pointer(&rate))

	// Create preprocessor state
	preprocState := C.speex_preprocess_state_init(C.int(frameSize), C.int(sampleRate))
	if preprocState == nil {
		C.speex_echo_state_destroy(echoState)
		return nil, errors.New("failed to create preprocessor state")
	}

	// Configure preprocessor
	val := C.int(1) // Enable denoising
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_DENOISE, unsafe.Pointer(&val))

	val = C.int(1) // Enable AGC
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_AGC, unsafe.Pointer(&val))

	// Link echo state to preprocessor
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_ECHO_STATE, unsafe.Pointer(echoState))

	return &AEC{
		echoState:    echoState,
		preprocState: preprocState,
		frameSize:    frameSize,
		filterLen:    filterLen,
	}, nil
}

// ProcessFrame processes a frame with both echo cancellation and noise suppression
func (aec *AEC) ProcessFrame(micFrame, speakerFrame []int16) []int16 {
	if len(micFrame) != aec.frameSize || len(speakerFrame) != aec.frameSize {
		return nil
	}

	output := make([]int16, aec.frameSize)
	micPtr := (*C.spx_int16_t)(unsafe.Pointer(&micFrame[0]))
	speakerPtr := (*C.spx_int16_t)(unsafe.Pointer(&speakerFrame[0]))
	outPtr := (*C.spx_int16_t)(unsafe.Pointer(&output[0]))

	// Apply echo cancellation
	C.speex_echo_cancellation(aec.echoState, micPtr, speakerPtr, outPtr)

	// Apply noise suppression and other preprocessing
	C.speex_preprocess_run(aec.preprocState, outPtr)

	return output
}

// ProcessFrameEchoOnly processes a frame with only echo cancellation (no noise suppression)
func (aec *AEC) ProcessFrameEchoOnly(micFrame, speakerFrame []int16) []int16 {
	if len(micFrame) != aec.frameSize || len(speakerFrame) != aec.frameSize {
		return nil
	}

	output := make([]int16, aec.frameSize)
	micPtr := (*C.spx_int16_t)(unsafe.Pointer(&micFrame[0]))
	speakerPtr := (*C.spx_int16_t)(unsafe.Pointer(&speakerFrame[0]))
	outPtr := (*C.spx_int16_t)(unsafe.Pointer(&output[0]))

	// Apply only echo cancellation
	C.speex_echo_cancellation(aec.echoState, micPtr, speakerPtr, outPtr)

	return output
}

// Reset resets the echo canceller state
func (aec *AEC) Reset() {
	if aec.echoState != nil {
		C.speex_echo_state_reset(aec.echoState)
	}
}

// Destroy cleans up resources
func (aec *AEC) Destroy() {
	if aec.preprocState != nil {
		C.speex_preprocess_state_destroy(aec.preprocState)
	}
	if aec.echoState != nil {
		C.speex_echo_state_destroy(aec.echoState)
	}
}
