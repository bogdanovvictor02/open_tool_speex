package types

// NSConfig holds noise suppression configuration parameters
type NSConfig struct {
	NoiseSuppress float64 // Noise suppression level in dB
	EnableVAD     bool    // Enable Voice Activity Detection
	VADProbStart  int     // VAD probability threshold for speech start (0-100)
	VADProbCont   int     // VAD probability threshold for speech continue (0-100)
	EnableAGC     bool    // Enable Automatic Gain Control
	AGCLevel      float64 // AGC target RMS level
}

// ProcessingMode represents the audio processing mode
type ProcessingMode int

const (
	ModeBypass ProcessingMode = iota
	ModeTestAlaw
	ModeNSOnly
	ModeAECOnly
	ModeNSFirst
	ModeAECFirst // default
)

// String returns the string representation of ProcessingMode
func (m ProcessingMode) String() string {
	switch m {
	case ModeBypass:
		return "BYPASS"
	case ModeTestAlaw:
		return "A-LAW-TEST"
	case ModeNSOnly:
		return "NS-only"
	case ModeAECOnly:
		return "AEC-only"
	case ModeNSFirst:
		return "NS-first"
	case ModeAECFirst:
		return "AEC-first"
	default:
		return "UNKNOWN"
	}
}

// Config holds the complete processing configuration
type Config struct {
	// File paths
	MicFile     string
	SpeakerFile string
	OutputFile  string

	// Processing mode
	Mode ProcessingMode

	// Processing parameters
	SampleRate     int
	FrameSize      int
	EchoTailMs     int
	FilterLen      int
	ProgressSec    float64
	UsePrevSpeaker bool

	// Noise suppression configuration
	NS NSConfig
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		OutputFile:     "output.alaw",
		Mode:           ModeAECFirst,
		SampleRate:     16000,
		FrameSize:      320,
		EchoTailMs:     200,
		ProgressSec:    16.0,
		UsePrevSpeaker: false,
		NS: NSConfig{
			NoiseSuppress: -15.0,
			EnableVAD:     false,
			VADProbStart:  80,
			VADProbCont:   65,
			EnableAGC:     false,
			AGCLevel:      30000.0,
		},
	}
}
