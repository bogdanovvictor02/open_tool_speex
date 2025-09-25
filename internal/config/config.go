package config

import (
	"flag"
	"fmt"
	"os"

	"open_tool_speex/pkg/types"
)

// ParseFlags parses command line flags and returns configuration
func ParseFlags() (*types.Config, error) {
	config := types.DefaultConfig()

	var (
		micFile        = flag.String("mic", "", "Path to microphone input file (raw A-law, 16kHz)")
		speakerFile    = flag.String("speaker", "", "Path to speaker reference file (raw A-law, 16kHz)")
		outputFile     = flag.String("output", config.OutputFile, "Path to output file (raw A-law, 16kHz)")
		usePrevSpeaker = flag.Bool("prev-speaker", config.UsePrevSpeaker, "Use previous speaker frame with current mic frame (delay compensation)")
		nsFirst        = flag.Bool("ns-first", false, "Apply Noise Suppression before Echo Cancellation (default: AEC then NS)")
		nsOnly         = flag.Bool("ns-only", false, "Apply only Noise Suppression (no echo cancellation)")
		aecOnly        = flag.Bool("aec-only", false, "Apply only Echo Cancellation (no noise suppression)")
		bypass         = flag.Bool("bypass", false, "Bypass all processing (copy input to output for testing)")
		testAlaw       = flag.Bool("test-alaw", false, "Test A-law encoding/decoding (A-law -> PCM -> A-law)")

		// Processing parameters (override defaults)
		sampleRate  = flag.Int("sample-rate", config.SampleRate, "Sample rate in Hz (e.g., 16000)")
		frameSize   = flag.Int("frame-size", config.FrameSize, "Frame size in samples (e.g., 320 for 20ms @16k)")
		echoTailMs  = flag.Int("echo-tail", config.EchoTailMs, "Echo tail length in milliseconds")
		filterLenIn = flag.Int("filter-len", 0, "Echo filter length in samples (override echo-tail if > 0)")
		progressSec = flag.Float64("progress-sec", config.ProgressSec, "Progress log interval in seconds (0 disables)")

		// Noise Suppression parameters
		noiseSuppress = flag.Float64("noise-suppress", config.NS.NoiseSuppress, "Noise suppression level in dB (more negative = more suppression)")
		enableVAD     = flag.Bool("vad", config.NS.EnableVAD, "Enable Voice Activity Detection")
		vadProbStart  = flag.Int("vad-prob-start", config.NS.VADProbStart, "VAD probability threshold for speech start (0-100)")
		vadProbCont   = flag.Int("vad-prob-continue", config.NS.VADProbCont, "VAD probability threshold for speech continue (0-100)")
		enableAGC     = flag.Bool("agc", config.NS.EnableAGC, "Enable Automatic Gain Control")
		agcLevel      = flag.Float64("agc-level", config.NS.AGCLevel, "AGC target RMS level")

		help = flag.Bool("help", false, "Show help")
	)

	flag.Parse()

	// Set file paths
	config.MicFile = *micFile
	config.SpeakerFile = *speakerFile
	config.OutputFile = *outputFile

	// Determine processing mode
	exclusiveCount := 0
	if *nsFirst {
		exclusiveCount++
		config.Mode = types.ModeNSFirst
	}
	if *nsOnly {
		exclusiveCount++
		config.Mode = types.ModeNSOnly
	}
	if *aecOnly {
		exclusiveCount++
		config.Mode = types.ModeAECOnly
	}
	if *bypass {
		exclusiveCount++
		config.Mode = types.ModeBypass
	}
	if *testAlaw {
		exclusiveCount++
		config.Mode = types.ModeTestAlaw
	}

	if exclusiveCount > 1 {
		return nil, fmt.Errorf("-ns-first, -ns-only, -aec-only, -bypass, and -test-alaw are mutually exclusive")
	}

	// Set processing parameters
	config.SampleRate = *sampleRate
	config.FrameSize = *frameSize
	config.EchoTailMs = *echoTailMs
	config.ProgressSec = *progressSec
	config.UsePrevSpeaker = *usePrevSpeaker

	// Derive filter length if not explicitly set
	config.FilterLen = *filterLenIn
	if config.FilterLen <= 0 {
		config.FilterLen = config.SampleRate * config.EchoTailMs / 1000
	}

	// Set noise suppression parameters
	config.NS.NoiseSuppress = *noiseSuppress
	config.NS.EnableVAD = *enableVAD
	config.NS.VADProbStart = *vadProbStart
	config.NS.VADProbCont = *vadProbCont
	config.NS.EnableAGC = *enableAGC
	config.NS.AGCLevel = *agcLevel

	// Validate configuration
	if err := validateConfig(&config, *help); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *types.Config, help bool) error {
	// Speaker file is required for all modes except NS-only, bypass, and test-alaw
	speakerRequired := config.Mode != types.ModeNSOnly && config.Mode != types.ModeBypass && config.Mode != types.ModeTestAlaw

	if help || config.MicFile == "" || (speakerRequired && config.SpeakerFile == "") {
		printHelp(config)
		if help || config.MicFile == "" || (speakerRequired && config.SpeakerFile == "") {
			return fmt.Errorf("missing required parameters")
		}
	}

	return nil
}

// printHelp prints the help message
func printHelp(config *types.Config) {
	fmt.Fprintf(os.Stderr, "Open Tool Speex\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s -mic <mic_file> [-speaker <speaker_file>] [-output <output_file>]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Parameters:\n")
	fmt.Fprintf(os.Stderr, "  -mic              Microphone input file (raw A-law, 16kHz mono)\n")
	fmt.Fprintf(os.Stderr, "  -speaker          Speaker reference file (raw A-law, 16kHz mono, required for AEC) \n")
	fmt.Fprintf(os.Stderr, "  -output           Output file (default: %s)\n", config.OutputFile)
	fmt.Fprintf(os.Stderr, "  -prev-speaker     Use previous speaker frame for delay compensation\n")
	fmt.Fprintf(os.Stderr, "  -ns-first         Apply Noise Suppression before Echo Cancellation\n")
	fmt.Fprintf(os.Stderr, "  -ns-only          Apply only Noise Suppression (no echo cancellation)\n")
	fmt.Fprintf(os.Stderr, "  -aec-only         Apply only Echo Cancellation (no noise suppression)\n")
	fmt.Fprintf(os.Stderr, "  -bypass           Bypass all processing (copy input to output for testing)\n")
	fmt.Fprintf(os.Stderr, "  -test-alaw        Test A-law encoding/decoding (A-law -> PCM -> A-law)\n\n")
	fmt.Fprintf(os.Stderr, "Processing Parameters:\n")
	fmt.Fprintf(os.Stderr, "  -sample-rate      Sample rate in Hz (default: %d)\n", config.SampleRate)
	fmt.Fprintf(os.Stderr, "  -frame-size       Frame size in samples (default: %d)\n", config.FrameSize)
	fmt.Fprintf(os.Stderr, "  -echo-tail        Echo tail length in ms (default: %d)\n", config.EchoTailMs)
	fmt.Fprintf(os.Stderr, "  -filter-len       Echo filter length in samples (override echo-tail if > 0)\n")
	fmt.Fprintf(os.Stderr, "  -progress-sec     Progress log interval in seconds (default: %.1f; 0 disables)\n\n", config.ProgressSec)
	fmt.Fprintf(os.Stderr, "Noise Suppression Settings:\n")
	fmt.Fprintf(os.Stderr, "  -noise-suppress   Noise suppression level in dB (default: %.1f, more negative = more suppression)\n", config.NS.NoiseSuppress)
	fmt.Fprintf(os.Stderr, "  -vad              Enable Voice Activity Detection\n")
	fmt.Fprintf(os.Stderr, "  -vad-prob-start   VAD probability threshold for speech start 0-100 (default: %d)\n", config.NS.VADProbStart)
	fmt.Fprintf(os.Stderr, "  -vad-prob-continue VAD probability threshold for speech continue 0-100 (default: %d)\n", config.NS.VADProbCont)
	fmt.Fprintf(os.Stderr, "  -agc              Enable Automatic Gain Control\n")
	fmt.Fprintf(os.Stderr, "  -agc-level        AGC target RMS level (default: %.1f)\n\n", config.NS.AGCLevel)
	fmt.Fprintf(os.Stderr, "  -help             Show this help\n\n")
	fmt.Fprintf(os.Stderr, "Frame size: %d samples (%.1fms)\n", config.FrameSize, float64(config.FrameSize)/float64(config.SampleRate)*1000)
	fmt.Fprintf(os.Stderr, "Echo tail: %dms (%d samples)\n", config.EchoTailMs, config.FilterLen)
	fmt.Fprintf(os.Stderr, "Processing: Echo cancellation + Noise suppression\n")
}
