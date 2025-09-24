package main

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

// SpeexAEC wraps Speex Echo Canceller and Preprocessor
type SpeexAEC struct {
	echoState    *C.SpeexEchoState
	preprocState *C.SpeexPreprocessState
	frameSize    int
	filterLen    int
}

// NewSpeexAEC creates new Speex AEC instance
// frameSize: samples per frame (320 for 20ms at 16kHz)
// filterLen: echo tail length in samples (3200 for 200ms at 16kHz)
// sampleRate: sample rate in Hz (16000)
func NewSpeexAEC(frameSize, filterLen, sampleRate int) (*SpeexAEC, error) {
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
		return nil, errors.New("failed to create preprocess state")
	}

	// Configure preprocessor for noise suppression only
	val := C.int(1)
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_DENOISE, unsafe.Pointer(&val))

	// Disable AGC
	val = C.int(0)
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_AGC, unsafe.Pointer(&val))

	// Associate echo state with preprocessor
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_ECHO_STATE, unsafe.Pointer(echoState))

	return &SpeexAEC{
		echoState:    echoState,
		preprocState: preprocState,
		frameSize:    frameSize,
		filterLen:    filterLen,
	}, nil
}

// ProcessFrame performs echo cancellation and noise suppression
// micFrame: microphone input (near-end)
// speakerFrame: speaker output (far-end/reference)
// Returns: processed output frame
func (aec *SpeexAEC) ProcessFrame(micFrame, speakerFrame []int16) []int16 {
	if len(micFrame) != aec.frameSize || len(speakerFrame) != aec.frameSize {
		return nil
	}

	output := make([]int16, aec.frameSize)

	// Convert Go slices to C arrays
	micPtr := (*C.spx_int16_t)(unsafe.Pointer(&micFrame[0]))
	speakerPtr := (*C.spx_int16_t)(unsafe.Pointer(&speakerFrame[0]))
	outPtr := (*C.spx_int16_t)(unsafe.Pointer(&output[0]))

	// Perform echo cancellation
	C.speex_echo_cancellation(aec.echoState, micPtr, speakerPtr, outPtr)

	// Apply noise suppression
	C.speex_preprocess_run(aec.preprocState, outPtr)

	return output
}

// ProcessFrameEchoOnly performs only echo cancellation (no noise suppression)
func (aec *SpeexAEC) ProcessFrameEchoOnly(micFrame, speakerFrame []int16) []int16 {
	if len(micFrame) != aec.frameSize || len(speakerFrame) != aec.frameSize {
		return nil
	}

	output := make([]int16, aec.frameSize)

	// Convert Go slices to C arrays
	micPtr := (*C.spx_int16_t)(unsafe.Pointer(&micFrame[0]))
	speakerPtr := (*C.spx_int16_t)(unsafe.Pointer(&speakerFrame[0]))
	outPtr := (*C.spx_int16_t)(unsafe.Pointer(&output[0]))

	// Perform only echo cancellation (skip noise suppression)
	C.speex_echo_cancellation(aec.echoState, micPtr, speakerPtr, outPtr)

	return output
}

// Reset resets the AEC state
func (aec *SpeexAEC) Reset() {
	if aec.echoState != nil {
		C.speex_echo_state_reset(aec.echoState)
	}
}

// Destroy cleans up resources
func (aec *SpeexAEC) Destroy() {
	if aec.preprocState != nil {
		C.speex_preprocess_state_destroy(aec.preprocState)
		aec.preprocState = nil
	}
	if aec.echoState != nil {
		C.speex_echo_state_destroy(aec.echoState)
		aec.echoState = nil
	}
}

// SpeexPreprocessor wraps standalone Speex Preprocessor (without echo state)
type SpeexPreprocessor struct {
	preprocState *C.SpeexPreprocessState
	frameSize    int
}

// NewSpeexPreprocessor creates new standalone Speex Preprocessor
func NewSpeexPreprocessor(frameSize, sampleRate int) (*SpeexPreprocessor, error) {
	if frameSize <= 0 || sampleRate <= 0 {
		return nil, errors.New("invalid parameters")
	}

	// Create preprocessor state
	preprocState := C.speex_preprocess_state_init(C.int(frameSize), C.int(sampleRate))
	if preprocState == nil {
		return nil, errors.New("failed to create preprocess state")
	}

	// Configure preprocessor for noise suppression only
	val := C.int(1)
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_DENOISE, unsafe.Pointer(&val))

	// Disable AGC
	val = C.int(0)
	C.speex_preprocess_ctl(preprocState, C.SPEEX_PREPROCESS_SET_AGC, unsafe.Pointer(&val))

	return &SpeexPreprocessor{
		preprocState: preprocState,
		frameSize:    frameSize,
	}, nil
}

// ProcessFrame applies noise suppression to input frame
func (ns *SpeexPreprocessor) ProcessFrame(inputFrame []int16) []int16 {
	if len(inputFrame) != ns.frameSize {
		return nil
	}

	output := make([]int16, ns.frameSize)
	copy(output, inputFrame)

	// Convert Go slice to C array
	outPtr := (*C.spx_int16_t)(unsafe.Pointer(&output[0]))

	// Apply noise suppression
	C.speex_preprocess_run(ns.preprocState, outPtr)

	return output
}

// Destroy cleans up preprocessor resources
func (ns *SpeexPreprocessor) Destroy() {
	if ns.preprocState != nil {
		C.speex_preprocess_state_destroy(ns.preprocState)
		ns.preprocState = nil
	}
}
