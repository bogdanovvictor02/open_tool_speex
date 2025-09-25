package processor

import (
	"fmt"
	"io"
	"os"

	"open_tool_speex/internal/audio"
	"open_tool_speex/internal/speex"
	"open_tool_speex/pkg/types"
)

// Processor handles audio processing
type Processor struct {
	config *types.Config
}

// NewProcessor creates a new audio processor
func NewProcessor(config *types.Config) *Processor {
	return &Processor{
		config: config,
	}
}

// Process performs audio processing based on the configuration
func (p *Processor) Process() error {
	// Open input files
	micFile, err := os.Open(p.config.MicFile)
	if err != nil {
		return fmt.Errorf("failed to open mic file: %w", err)
	}
	defer micFile.Close()

	var speakerFile *os.File
	if p.needsSpeakerFile() {
		speakerFile, err = os.Open(p.config.SpeakerFile)
		if err != nil {
			return fmt.Errorf("failed to open speaker file: %w", err)
		}
		defer speakerFile.Close()
	}

	// Create output file
	outFile, err := os.Create(p.config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Initialize components based on mode
	var aec *speex.AEC
	var separateNS *speex.Preprocessor

	if err := p.initializeComponents(&aec, &separateNS); err != nil {
		return err
	}

	// Cleanup components
	if aec != nil {
		defer aec.Destroy()
	}
	if separateNS != nil {
		defer separateNS.Destroy()
	}

	// Process audio
	return p.processAudio(micFile, speakerFile, outFile, aec, separateNS)
}

// needsSpeakerFile returns true if speaker file is needed for current mode
func (p *Processor) needsSpeakerFile() bool {
	return p.config.Mode != types.ModeNSOnly && 
		   p.config.Mode != types.ModeBypass && 
		   p.config.Mode != types.ModeTestAlaw
}

// initializeComponents initializes AEC and preprocessor based on mode
func (p *Processor) initializeComponents(aec **speex.AEC, separateNS **speex.Preprocessor) error {
	switch p.config.Mode {
	case types.ModeBypass, types.ModeTestAlaw:
		// No processing needed
		return nil

	case types.ModeNSOnly:
		// Only need standalone preprocessor
		ns, err := speex.NewPreprocessorWithConfig(p.config.FrameSize, p.config.SampleRate, p.config.NS)
		if err != nil {
			return fmt.Errorf("failed to initialize NS: %w", err)
		}
		*separateNS = ns

	case types.ModeAECOnly:
		// Only need AEC
		aecInstance, err := speex.NewAEC(p.config.FrameSize, p.config.FilterLen, p.config.SampleRate)
		if err != nil {
			return fmt.Errorf("failed to initialize AEC: %w", err)
		}
		*aec = aecInstance

	case types.ModeNSFirst:
		// Need both AEC and separate preprocessor
		aecInstance, err := speex.NewAEC(p.config.FrameSize, p.config.FilterLen, p.config.SampleRate)
		if err != nil {
			return fmt.Errorf("failed to initialize AEC: %w", err)
		}
		*aec = aecInstance

		ns, err := speex.NewPreprocessorWithConfig(p.config.FrameSize, p.config.SampleRate, p.config.NS)
		if err != nil {
			return fmt.Errorf("failed to initialize separate NS: %w", err)
		}
		*separateNS = ns

	case types.ModeAECFirst:
		// Default mode: AEC with built-in preprocessor
		aecInstance, err := speex.NewAEC(p.config.FrameSize, p.config.FilterLen, p.config.SampleRate)
		if err != nil {
			return fmt.Errorf("failed to initialize AEC: %w", err)
		}
		*aec = aecInstance

	default:
		return fmt.Errorf("unknown processing mode: %v", p.config.Mode)
	}

	return nil
}

// processAudio performs the main audio processing loop
func (p *Processor) processAudio(micFile, speakerFile, outFile *os.File, aec *speex.AEC, separateNS *speex.Preprocessor) error {
	// Processing buffers
	micAlawFrame := make([]byte, p.config.FrameSize)
	speakerAlawFrame := make([]byte, p.config.FrameSize)
	micPcmFrame := make([]int16, p.config.FrameSize)
	speakerPcmFrame := make([]int16, p.config.FrameSize)

	// Previous speaker frame for delay compensation
	var prevSpeakerPcmFrame []int16
	if p.config.UsePrevSpeaker {
		prevSpeakerPcmFrame = make([]int16, p.config.FrameSize)
	}

	frameCount := 0

	// Print processing mode info
	p.printModeInfo()

	// Main processing loop
	for {
		// Read mic frame
		micBytesRead, err := io.ReadFull(micFile, micAlawFrame)
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("error reading mic file: %w", err)
		}

		// Read speaker frame (only for AEC modes)
		var speakerBytesRead int
		if p.needsSpeakerFile() {
			speakerBytesRead, err = io.ReadFull(speakerFile, speakerAlawFrame)
			if err == io.EOF {
				break
			}
			if err != nil && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("error reading speaker file: %w", err)
			}
		}

		// Handle partial frames at end of file
		p.zeroPadFrames(micAlawFrame, micBytesRead, speakerAlawFrame, speakerBytesRead)

		// Convert A-law to PCM16
		audio.AlawBufferToPCM16(micAlawFrame, micPcmFrame)
		if p.needsSpeakerFile() {
			audio.AlawBufferToPCM16(speakerAlawFrame, speakerPcmFrame)
		}

		// Process frame based on mode
		outputAlawFrame, err := p.processFrame(micPcmFrame, speakerPcmFrame, prevSpeakerPcmFrame, aec, separateNS)
		if err != nil {
			return fmt.Errorf("error processing frame %d: %w", frameCount, err)
		}

		// Update previous speaker frame for next iteration
		if p.config.UsePrevSpeaker && p.needsSpeakerFile() {
			copy(prevSpeakerPcmFrame, speakerPcmFrame)
		}

		// Write output frame
		if _, err := outFile.Write(outputAlawFrame); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}

		frameCount++
		p.logProgress(frameCount)
	}

	duration := float64(frameCount*p.config.FrameSize) / float64(p.config.SampleRate)
	fmt.Printf("Total processed: %.1f seconds (%d frames)\n", duration, frameCount)

	return nil
}

// processFrame processes a single frame based on the current mode
func (p *Processor) processFrame(micPcmFrame, speakerPcmFrame, prevSpeakerPcmFrame []int16, aec *speex.AEC, separateNS *speex.Preprocessor) ([]byte, error) {
	switch p.config.Mode {
	case types.ModeBypass:
		// Bypass mode: no processing, copy A-law input directly to output
		// Note: This should copy from micAlawFrame, but we need to pass it as parameter
		// For now, convert PCM back to A-law
		outputAlawFrame := make([]byte, p.config.FrameSize)
		audio.PCM16BufferToAlaw(micPcmFrame, outputAlawFrame)
		return outputAlawFrame, nil

	case types.ModeTestAlaw:
		// Test A-law mode: A-law -> PCM -> A-law
		outputAlawFrame := make([]byte, p.config.FrameSize)
		audio.PCM16BufferToAlaw(micPcmFrame, outputAlawFrame)
		return outputAlawFrame, nil

	case types.ModeNSOnly:
		// NS-only mode: only noise suppression
		outputPcmFrame := separateNS.ProcessFrame(micPcmFrame)
		if outputPcmFrame == nil {
			return nil, fmt.Errorf("NS processing failed")
		}
		outputAlawFrame := make([]byte, p.config.FrameSize)
		audio.PCM16BufferToAlaw(outputPcmFrame, outputAlawFrame)
		return outputAlawFrame, nil

	case types.ModeAECOnly:
		// AEC-only mode: only echo cancellation
		aecSpeakerFrame := p.getAECSpeakerFrame(speakerPcmFrame, prevSpeakerPcmFrame)
		outputPcmFrame := aec.ProcessFrameEchoOnly(micPcmFrame, aecSpeakerFrame)
		if outputPcmFrame == nil {
			return nil, fmt.Errorf("AEC processing failed")
		}
		outputAlawFrame := make([]byte, p.config.FrameSize)
		audio.PCM16BufferToAlaw(outputPcmFrame, outputAlawFrame)
		return outputAlawFrame, nil

	case types.ModeNSFirst:
		// NS-first mode: noise suppression, then echo cancellation
		nsOutput := separateNS.ProcessFrame(micPcmFrame)
		if nsOutput == nil {
			return nil, fmt.Errorf("NS processing failed")
		}
		aecSpeakerFrame := p.getAECSpeakerFrame(speakerPcmFrame, prevSpeakerPcmFrame)
		outputPcmFrame := aec.ProcessFrameEchoOnly(nsOutput, aecSpeakerFrame)
		if outputPcmFrame == nil {
			return nil, fmt.Errorf("AEC processing failed")
		}
		outputAlawFrame := make([]byte, p.config.FrameSize)
		audio.PCM16BufferToAlaw(outputPcmFrame, outputAlawFrame)
		return outputAlawFrame, nil

	case types.ModeAECFirst:
		// AEC-first mode: echo cancellation, then noise suppression (default)
		aecSpeakerFrame := p.getAECSpeakerFrame(speakerPcmFrame, prevSpeakerPcmFrame)
		outputPcmFrame := aec.ProcessFrame(micPcmFrame, aecSpeakerFrame)
		if outputPcmFrame == nil {
			return nil, fmt.Errorf("AEC processing failed")
		}
		outputAlawFrame := make([]byte, p.config.FrameSize)
		audio.PCM16BufferToAlaw(outputPcmFrame, outputAlawFrame)
		return outputAlawFrame, nil

	default:
		return nil, fmt.Errorf("unknown processing mode: %v", p.config.Mode)
	}
}

// getAECSpeakerFrame returns the appropriate speaker frame for AEC
func (p *Processor) getAECSpeakerFrame(speakerPcmFrame, prevSpeakerPcmFrame []int16) []int16 {
	if p.config.UsePrevSpeaker {
		return prevSpeakerPcmFrame
	}
	return speakerPcmFrame
}

// zeroPadFrames zero-pads partial frames at end of file
func (p *Processor) zeroPadFrames(micAlawFrame []byte, micBytesRead int, speakerAlawFrame []byte, speakerBytesRead int) {
	if micBytesRead < p.config.FrameSize {
		for i := micBytesRead; i < p.config.FrameSize; i++ {
			micAlawFrame[i] = 0xD5 // A-law silence
		}
	}
	if p.needsSpeakerFile() && speakerBytesRead < p.config.FrameSize {
		for i := speakerBytesRead; i < p.config.FrameSize; i++ {
			speakerAlawFrame[i] = 0xD5 // A-law silence
		}
	}
}

// printModeInfo prints information about the processing mode
func (p *Processor) printModeInfo() {
	var modeStr []string
	modeStr = append(modeStr, p.config.Mode.String())
	
	if p.config.UsePrevSpeaker && p.needsSpeakerFile() {
		modeStr = append(modeStr, "delay compensation")
	}

	if len(modeStr) > 0 {
		fmt.Printf("Processing audio frames (size: %d samples, %.1fms) with %s...\n",
			p.config.FrameSize, float64(p.config.FrameSize)/float64(p.config.SampleRate)*1000,
			fmt.Sprintf("%v", modeStr))
	} else {
		fmt.Printf("Processing audio frames (size: %d samples, %.1fms)...\n",
			p.config.FrameSize, float64(p.config.FrameSize)/float64(p.config.SampleRate)*1000)
	}

	if p.config.Mode == types.ModeTestAlaw {
		fmt.Printf("A-law test mode: Testing A-law -> PCM -> A-law conversion chain\n")
	}
}

// logProgress logs processing progress
func (p *Processor) logProgress(frameCount int) {
	if p.config.ProgressSec > 0 {
		framesPerInterval := int(float64(p.config.SampleRate)/float64(p.config.FrameSize)*p.config.ProgressSec + 0.5)
		if framesPerInterval <= 0 {
			framesPerInterval = 1
		}
		if frameCount%framesPerInterval == 0 {
			duration := float64(frameCount*p.config.FrameSize) / float64(p.config.SampleRate)
			fmt.Printf("Processed %.1f seconds (%d frames)\n", duration, frameCount)
		}
	}
}
