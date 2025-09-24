package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

// Default processing parameters (can be overridden by CLI flags)
const (
	defaultSampleRate = 16000
	defaultFrameSize  = 320 // 20ms at 16kHz
	defaultEchoTailMs = 200 // 200ms echo tail in milliseconds
)

func main() {
	var (
		micFile        = flag.String("mic", "", "Path to microphone input file (raw A-law, 16kHz)")
		speakerFile    = flag.String("speaker", "", "Path to speaker reference file (raw A-law, 16kHz)")
		outputFile     = flag.String("output", "output.alaw", "Path to output file (raw A-law, 16kHz)")
		usePrevSpeaker = flag.Bool("prev-speaker", false, "Use previous speaker frame with current mic frame (delay compensation)")
		nsFirst        = flag.Bool("ns-first", false, "Apply Noise Suppression before Echo Cancellation (default: AEC then NS)")
		nsOnly         = flag.Bool("ns-only", false, "Apply only Noise Suppression (no echo cancellation)")
		aecOnly        = flag.Bool("aec-only", false, "Apply only Echo Cancellation (no noise suppression)")
		bypass         = flag.Bool("bypass", false, "Bypass all processing (copy input to output for testing)")

		// Processing parameters (override defaults)
		sampleRate  = flag.Int("sample-rate", defaultSampleRate, "Sample rate in Hz (e.g., 16000)")
		frameSize   = flag.Int("frame-size", defaultFrameSize, "Frame size in samples (e.g., 320 for 20ms @16k)")
		echoTailMs  = flag.Int("echo-tail", defaultEchoTailMs, "Echo tail length in milliseconds")
		filterLenIn = flag.Int("filter-len", 0, "Echo filter length in samples (override echo-tail if > 0)")
		progressSec = flag.Float64("progress-sec", 16.0, "Progress log interval in seconds (0 disables)")

		// Noise Suppression parameters
		noiseSuppress = flag.Float64("noise-suppress", -15.0, "Noise suppression level in dB (more negative = more suppression)")
		enableVAD     = flag.Bool("vad", false, "Enable Voice Activity Detection")
		vadProbStart  = flag.Int("vad-prob-start", 80, "VAD probability threshold for speech start (0-100)")
		vadProbCont   = flag.Int("vad-prob-continue", 65, "VAD probability threshold for speech continue (0-100)")
		enableAGC     = flag.Bool("agc", false, "Enable Automatic Gain Control")
		agcLevel      = flag.Float64("agc-level", 30000.0, "AGC target RMS level")

		help = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	// Derive filter length if not explicitly set
	effectiveFilterLen := *filterLenIn
	if effectiveFilterLen <= 0 {
		effectiveFilterLen = *sampleRate * *echoTailMs / 1000
	}

	// Validate mutually exclusive options
	exclusiveCount := 0
	if *nsFirst {
		exclusiveCount++
	}
	if *nsOnly {
		exclusiveCount++
	}
	if *aecOnly {
		exclusiveCount++
	}

	if *bypass {
		exclusiveCount++ // bypass is also exclusive
	}

	if exclusiveCount > 1 {
		fmt.Fprintf(os.Stderr, "Error: -ns-first, -ns-only, -aec-only, and -bypass are mutually exclusive\n")
		os.Exit(1)
	}

	// Speaker file is required for all modes except NS-only and bypass
	speakerRequired := !*nsOnly && !*bypass
	if *help || *micFile == "" || (speakerRequired && *speakerFile == "") {
		fmt.Fprintf(os.Stderr, "Speex AEC Console Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s -mic <mic_file> [-speaker <speaker_file>] [-output <output_file>]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Parameters:\n")
		fmt.Fprintf(os.Stderr, "  -mic              Microphone input file (raw A-law, 16kHz mono)\n")
		fmt.Fprintf(os.Stderr, "  -speaker          Speaker reference file (raw A-law, 16kHz mono, required for AEC) \n")
		fmt.Fprintf(os.Stderr, "  -output           Output file (default: output.alaw)\n")
		fmt.Fprintf(os.Stderr, "  -prev-speaker     Use previous speaker frame for delay compensation\n")
		fmt.Fprintf(os.Stderr, "  -ns-first         Apply Noise Suppression before Echo Cancellation\n")
		fmt.Fprintf(os.Stderr, "  -ns-only          Apply only Noise Suppression (no echo cancellation)\n")
		fmt.Fprintf(os.Stderr, "  -aec-only         Apply only Echo Cancellation (no noise suppression)\n")
		fmt.Fprintf(os.Stderr, "  -bypass           Bypass all processing (copy input to output for testing)\n\n")
		fmt.Fprintf(os.Stderr, "Processing Parameters:\n")
		fmt.Fprintf(os.Stderr, "  -sample-rate      Sample rate in Hz (default: %d)\n", defaultSampleRate)
		fmt.Fprintf(os.Stderr, "  -frame-size       Frame size in samples (default: %d)\n", defaultFrameSize)
		fmt.Fprintf(os.Stderr, "  -echo-tail        Echo tail length in ms (default: %d)\n", defaultEchoTailMs)
		fmt.Fprintf(os.Stderr, "  -filter-len       Echo filter length in samples (override echo-tail if > 0)\n")
		fmt.Fprintf(os.Stderr, "  -progress-sec     Progress log interval in seconds (default: %.1f; 0 disables)\n\n", *progressSec)
		fmt.Fprintf(os.Stderr, "Noise Suppression Settings:\n")
		fmt.Fprintf(os.Stderr, "  -noise-suppress   Noise suppression level in dB (default: -15.0, more negative = more suppression)\n")
		fmt.Fprintf(os.Stderr, "  -vad              Enable Voice Activity Detection\n")
		fmt.Fprintf(os.Stderr, "  -vad-prob-start   VAD probability threshold for speech start 0-100 (default: 80)\n")
		fmt.Fprintf(os.Stderr, "  -vad-prob-continue VAD probability threshold for speech continue 0-100 (default: 65)\n")
		fmt.Fprintf(os.Stderr, "  -agc              Enable Automatic Gain Control\n")
		fmt.Fprintf(os.Stderr, "  -agc-level        AGC target RMS level (default: 30000.0)\n\n")
		fmt.Fprintf(os.Stderr, "  -help             Show this help\n\n")
		fmt.Fprintf(os.Stderr, "Frame size: %d samples (%.1fms)\n", *frameSize, float64(*frameSize)/float64(*sampleRate)*1000)
		fmt.Fprintf(os.Stderr, "Echo tail: %dms (%d samples)\n", *echoTailMs, effectiveFilterLen)
		fmt.Fprintf(os.Stderr, "Processing: Echo cancellation + Noise suppression\n")
		os.Exit(1)
	}

	// Create NS configuration
	nsConfig := NSConfig{
		NoiseSuppress: *noiseSuppress,
		EnableVAD:     *enableVAD,
		VADProbStart:  *vadProbStart,
		VADProbCont:   *vadProbCont,
		EnableAGC:     *enableAGC,
		AGCLevel:      *agcLevel,
	}

	if err := processAEC(*micFile, *speakerFile, *outputFile, *usePrevSpeaker, *nsFirst, *nsOnly, *aecOnly, *bypass, nsConfig, *sampleRate, *frameSize, effectiveFilterLen, *progressSec); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("AEC processing completed: %s -> %s\n", *micFile, *outputFile)
}

// processAEC performs echo cancellation on input files
func processAEC(micPath, speakerPath, outputPath string, usePrevSpeaker, nsFirst, nsOnly, aecOnly, bypass bool, nsConfig NSConfig, sampleRate, frameSize, filterLen int, progressSec float64) error {
	// Open input files
	micFile, err := os.Open(micPath)
	if err != nil {
		return fmt.Errorf("failed to open mic file: %w", err)
	}
	defer micFile.Close()

	var speakerFile *os.File
	if !nsOnly && !bypass { // Speaker file needed for AEC modes (not for NS-only or bypass)
		speakerFile, err = os.Open(speakerPath)
		if err != nil {
			return fmt.Errorf("failed to open speaker file: %w", err)
		}
		defer speakerFile.Close()
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Initialize components based on mode
	var aec *SpeexAEC
	var separateNS *SpeexPreprocessor

	if bypass {
		// Bypass mode: no processing needed
		// Skip initialization
	} else if nsOnly {
		// NS-only mode: only need standalone preprocessor
		separateNS, err = NewSpeexPreprocessorWithConfig(frameSize, sampleRate, nsConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize NS: %w", err)
		}
		defer separateNS.Destroy()
	} else {
		// AEC modes (including aec-only): need AEC
		aec, err = NewSpeexAEC(frameSize, filterLen, sampleRate)
		if err != nil {
			return fmt.Errorf("failed to initialize AEC: %w", err)
		}
		defer aec.Destroy()

		// For NS-first mode, we need separate preprocessor that's not tied to echo state
		// (AEC-only mode doesn't need separate NS)
		if nsFirst {
			separateNS, err = NewSpeexPreprocessorWithConfig(frameSize, sampleRate, nsConfig)
			if err != nil {
				return fmt.Errorf("failed to initialize separate NS: %w", err)
			}
			defer separateNS.Destroy()
		}
	}

	// Processing buffers
	micAlawFrame := make([]byte, frameSize)
	speakerAlawFrame := make([]byte, frameSize)
	micPcmFrame := make([]int16, frameSize)
	speakerPcmFrame := make([]int16, frameSize)

	// Previous speaker frame for delay compensation
	var prevSpeakerPcmFrame []int16
	if usePrevSpeaker {
		prevSpeakerPcmFrame = make([]int16, frameSize)
		// Initialize with silence
		for i := range prevSpeakerPcmFrame {
			prevSpeakerPcmFrame[i] = 0
		}
	}

	frameCount := 0

	// Print processing mode info
	var modeStr []string
	if bypass {
		modeStr = append(modeStr, "BYPASS")
	} else if nsOnly {
		modeStr = append(modeStr, "NS-only")
	} else if aecOnly {
		modeStr = append(modeStr, "AEC-only")
	} else if nsFirst {
		modeStr = append(modeStr, "NS-first")
	} else {
		modeStr = append(modeStr, "AEC-first")
	}
	if usePrevSpeaker && !nsOnly && !bypass {
		modeStr = append(modeStr, "delay compensation")
	}

	if len(modeStr) > 0 {
		fmt.Printf("Processing audio frames (size: %d samples, %.1fms) with %s...\n",
			frameSize, float64(frameSize)/float64(sampleRate)*1000,
			fmt.Sprintf("%v", modeStr))
	} else {
		fmt.Printf("Processing audio frames (size: %d samples, %.1fms)...\n",
			frameSize, float64(frameSize)/float64(sampleRate)*1000)
	}

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
		if !nsOnly && !bypass {
			speakerBytesRead, err = io.ReadFull(speakerFile, speakerAlawFrame)
			if err == io.EOF {
				break
			}
			if err != nil && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("error reading speaker file: %w", err)
			}
		}

		// Handle partial frames at end of file
		if micBytesRead < frameSize {
			// Zero-pad partial mic frames
			for i := micBytesRead; i < frameSize; i++ {
				micAlawFrame[i] = 0xD5 // A-law silence
			}
		}
		if !nsOnly && !bypass && speakerBytesRead < frameSize {
			// Zero-pad partial speaker frames (only for AEC modes)
			for i := speakerBytesRead; i < frameSize; i++ {
				speakerAlawFrame[i] = 0xD5 // A-law silence
			}
		}

		// Convert A-law to PCM16
		AlawBufferToPCM16(micAlawFrame, micPcmFrame)
		if !nsOnly && !bypass {
			AlawBufferToPCM16(speakerAlawFrame, speakerPcmFrame)
		}

		// Choose which speaker frame to use for AEC
		var aecSpeakerFrame []int16
		if usePrevSpeaker {
			// Use previous speaker frame with current mic frame
			aecSpeakerFrame = prevSpeakerPcmFrame
		} else {
			// Use current speaker frame with current mic frame
			aecSpeakerFrame = speakerPcmFrame
		}

		// Perform processing based on selected mode
		var outputAlawFrame []byte
		if bypass {
			// Mode 0: Bypass (no processing, copy A-law input directly to output)
			outputAlawFrame = make([]byte, frameSize)
			copy(outputAlawFrame, micAlawFrame)
		} else {
			// All other modes need PCM16 processing
			var outputPcmFrame []int16
			if nsOnly {
				// Mode 1: NS only (no echo cancellation)
				outputPcmFrame = separateNS.ProcessFrame(micPcmFrame)
				if outputPcmFrame == nil {
					return fmt.Errorf("NS processing failed at frame %d", frameCount)
				}
			} else if aecOnly {
				// Mode 2: AEC only (no noise suppression)
				outputPcmFrame = aec.ProcessFrameEchoOnly(micPcmFrame, aecSpeakerFrame)
				if outputPcmFrame == nil {
					return fmt.Errorf("AEC processing failed at frame %d", frameCount)
				}
			} else if nsFirst {
				// Mode 3: NS first, then AEC
				// Apply noise suppression to microphone signal first
				nsOutput := separateNS.ProcessFrame(micPcmFrame)
				if nsOutput == nil {
					return fmt.Errorf("NS processing failed at frame %d", frameCount)
				}
				// Then apply echo cancellation
				outputPcmFrame = aec.ProcessFrameEchoOnly(nsOutput, aecSpeakerFrame)
				if outputPcmFrame == nil {
					return fmt.Errorf("AEC processing failed at frame %d", frameCount)
				}
			} else {
				// Mode 4: AEC first, then NS (default)
				outputPcmFrame = aec.ProcessFrame(micPcmFrame, aecSpeakerFrame)
				if outputPcmFrame == nil {
					return fmt.Errorf("AEC processing failed at frame %d", frameCount)
				}
			}

			// Convert PCM16 back to A-law for non-bypass modes
			outputAlawFrame = make([]byte, frameSize)
			PCM16BufferToAlaw(outputPcmFrame, outputAlawFrame)
		}

		// Update previous speaker frame for next iteration
		if usePrevSpeaker && !bypass {
			copy(prevSpeakerPcmFrame, speakerPcmFrame)
		}

		// Write output frame
		if _, err := outFile.Write(outputAlawFrame); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}

		frameCount++
		// Dynamic progress interval based on flags
		if progressSec > 0 {
			framesPerInterval := int(float64(sampleRate)/float64(frameSize)*progressSec + 0.5)
			if framesPerInterval <= 0 {
				framesPerInterval = 1
			}
			if frameCount%framesPerInterval == 0 {
				duration := float64(frameCount*frameSize) / float64(sampleRate)
				fmt.Printf("Processed %.1f seconds (%d frames)\n", duration, frameCount)
			}
		}
	}

	duration := float64(frameCount*frameSize) / float64(sampleRate)
	fmt.Printf("Total processed: %.1f seconds (%d frames)\n", duration, frameCount)

	return nil
}
