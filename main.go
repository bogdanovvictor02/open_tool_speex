package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	SAMPLE_RATE = 16000
	FRAME_SIZE  = 320                            // 20ms at 16kHz
	ECHO_TAIL   = 200                            // 200ms echo tail in milliseconds
	FILTER_LEN  = SAMPLE_RATE * ECHO_TAIL / 1000 // Convert to samples
)

func main() {
	var (
		micFile        = flag.String("mic", "", "Path to microphone input file (raw A-law, 16kHz)")
		speakerFile    = flag.String("speaker", "", "Path to speaker reference file (raw A-law, 16kHz)")
		outputFile     = flag.String("output", "output.alaw", "Path to output file (raw A-law, 16kHz)")
		usePrevSpeaker = flag.Bool("prev-speaker", false, "Use previous speaker frame with current mic frame (delay compensation)")
		nsFirst        = flag.Bool("ns-first", false, "Apply Noise Suppression before Echo Cancellation (default: AEC then NS)")
		nsOnly         = flag.Bool("ns-only", false, "Apply only Noise Suppression (no echo cancellation)")
		help           = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	// Validate mutually exclusive options
	if *nsFirst && *nsOnly {
		fmt.Fprintf(os.Stderr, "Error: -ns-first and -ns-only are mutually exclusive\n")
		os.Exit(1)
	}

	// Speaker file is optional for NS-only mode
	speakerRequired := !*nsOnly
	if *help || *micFile == "" || (speakerRequired && *speakerFile == "") {
		fmt.Fprintf(os.Stderr, "Speex AEC Console Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s -mic <mic_file> [-speaker <speaker_file>] [-output <output_file>]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Parameters:\n")
		fmt.Fprintf(os.Stderr, "  -mic           Microphone input file (raw A-law, 16kHz mono)\n")
		fmt.Fprintf(os.Stderr, "  -speaker       Speaker reference file (raw A-law, 16kHz mono, required for AEC) \n")
		fmt.Fprintf(os.Stderr, "  -output        Output file (default: output.alaw)\n")
		fmt.Fprintf(os.Stderr, "  -prev-speaker  Use previous speaker frame for delay compensation\n")
		fmt.Fprintf(os.Stderr, "  -ns-first      Apply Noise Suppression before Echo Cancellation\n")
		fmt.Fprintf(os.Stderr, "  -ns-only       Apply only Noise Suppression (no echo cancellation)\n")
		fmt.Fprintf(os.Stderr, "  -help          Show this help\n\n")
		fmt.Fprintf(os.Stderr, "Frame size: %d samples (20ms)\n", FRAME_SIZE)
		fmt.Fprintf(os.Stderr, "Echo tail: %dms (%d samples)\n", ECHO_TAIL, FILTER_LEN)
		fmt.Fprintf(os.Stderr, "Processing: Echo cancellation + Noise suppression\n")
		os.Exit(1)
	}

	if err := processAEC(*micFile, *speakerFile, *outputFile, *usePrevSpeaker, *nsFirst, *nsOnly); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("AEC processing completed: %s -> %s\n", *micFile, *outputFile)
}

// processAEC performs echo cancellation on input files
func processAEC(micPath, speakerPath, outputPath string, usePrevSpeaker, nsFirst, nsOnly bool) error {
	// Open input files
	micFile, err := os.Open(micPath)
	if err != nil {
		return fmt.Errorf("failed to open mic file: %w", err)
	}
	defer micFile.Close()

	var speakerFile *os.File
	if !nsOnly {
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

	if nsOnly {
		// NS-only mode: only need standalone preprocessor
		separateNS, err = NewSpeexPreprocessor(FRAME_SIZE, SAMPLE_RATE)
		if err != nil {
			return fmt.Errorf("failed to initialize NS: %w", err)
		}
		defer separateNS.Destroy()
	} else {
		// AEC modes: need AEC
		aec, err = NewSpeexAEC(FRAME_SIZE, FILTER_LEN, SAMPLE_RATE)
		if err != nil {
			return fmt.Errorf("failed to initialize AEC: %w", err)
		}
		defer aec.Destroy()

		// For NS-first mode, we need separate preprocessor that's not tied to echo state
		if nsFirst {
			separateNS, err = NewSpeexPreprocessor(FRAME_SIZE, SAMPLE_RATE)
			if err != nil {
				return fmt.Errorf("failed to initialize separate NS: %w", err)
			}
			defer separateNS.Destroy()
		}
	}

	// Processing buffers
	micAlawFrame := make([]byte, FRAME_SIZE)
	speakerAlawFrame := make([]byte, FRAME_SIZE)
	micPcmFrame := make([]int16, FRAME_SIZE)
	speakerPcmFrame := make([]int16, FRAME_SIZE)
	outputAlawFrame := make([]byte, FRAME_SIZE)

	// Previous speaker frame for delay compensation
	var prevSpeakerPcmFrame []int16
	if usePrevSpeaker {
		prevSpeakerPcmFrame = make([]int16, FRAME_SIZE)
		// Initialize with silence
		for i := range prevSpeakerPcmFrame {
			prevSpeakerPcmFrame[i] = 0
		}
	}

	frameCount := 0

	// Print processing mode info
	var modeStr []string
	if nsOnly {
		modeStr = append(modeStr, "NS-only")
	} else if nsFirst {
		modeStr = append(modeStr, "NS-first")
	} else {
		modeStr = append(modeStr, "AEC-first")
	}
	if usePrevSpeaker && !nsOnly {
		modeStr = append(modeStr, "delay compensation")
	}

	if len(modeStr) > 0 {
		fmt.Printf("Processing audio frames (size: %d samples, %.1fms) with %s...\n",
			FRAME_SIZE, float64(FRAME_SIZE)/float64(SAMPLE_RATE)*1000,
			fmt.Sprintf("%v", modeStr))
	} else {
		fmt.Printf("Processing audio frames (size: %d samples, %.1fms)...\n",
			FRAME_SIZE, float64(FRAME_SIZE)/float64(SAMPLE_RATE)*1000)
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
		if !nsOnly {
			speakerBytesRead, err = io.ReadFull(speakerFile, speakerAlawFrame)
			if err == io.EOF {
				break
			}
			if err != nil && err != io.ErrUnexpectedEOF {
				return fmt.Errorf("error reading speaker file: %w", err)
			}
		}

		// Handle partial frames at end of file
		if micBytesRead < FRAME_SIZE {
			// Zero-pad partial mic frames
			for i := micBytesRead; i < FRAME_SIZE; i++ {
				micAlawFrame[i] = 0xD5 // A-law silence
			}
		}
		if !nsOnly && speakerBytesRead < FRAME_SIZE {
			// Zero-pad partial speaker frames (only for AEC modes)
			for i := speakerBytesRead; i < FRAME_SIZE; i++ {
				speakerAlawFrame[i] = 0xD5 // A-law silence
			}
		}

		// Convert A-law to PCM16
		AlawBufferToPCM16(micAlawFrame, micPcmFrame)
		if !nsOnly {
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
		var outputPcmFrame []int16
		if nsOnly {
			// Mode 1: NS only (no echo cancellation)
			outputPcmFrame = separateNS.ProcessFrame(micPcmFrame)
			if outputPcmFrame == nil {
				return fmt.Errorf("NS processing failed at frame %d", frameCount)
			}
		} else if nsFirst {
			// Mode 2: NS first, then AEC
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
			// Mode 3: AEC first, then NS (default)
			outputPcmFrame = aec.ProcessFrame(micPcmFrame, aecSpeakerFrame)
			if outputPcmFrame == nil {
				return fmt.Errorf("AEC processing failed at frame %d", frameCount)
			}
		}

		// Update previous speaker frame for next iteration
		if usePrevSpeaker {
			copy(prevSpeakerPcmFrame, speakerPcmFrame)
		}

		// Convert PCM16 back to A-law
		PCM16BufferToAlaw(outputPcmFrame, outputAlawFrame)

		// Write output frame
		if _, err := outFile.Write(outputAlawFrame); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}

		frameCount++
		if frameCount%800 == 0 { // Progress every ~16 seconds at 16kHz
			duration := float64(frameCount*FRAME_SIZE) / float64(SAMPLE_RATE)
			fmt.Printf("Processed %.1f seconds (%d frames)\n", duration, frameCount)
		}
	}

	duration := float64(frameCount*FRAME_SIZE) / float64(SAMPLE_RATE)
	fmt.Printf("Total processed: %.1f seconds (%d frames)\n", duration, frameCount)

	return nil
}
