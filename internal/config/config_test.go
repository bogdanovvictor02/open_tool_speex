package config

import (
	"flag"
	"os"
	"testing"

	"open_tool_speex/pkg/types"
)

func TestParseFlags(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(*types.Config) bool
	}{
		{
			name: "valid config with all flags",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-speaker", "ref.alaw",
				"-output", "out.alaw",
				"-sample-rate", "16000",
				"-frame-size", "320",
				"-echo-tail", "200",
				"-noise-suppress", "-20.0",
				"-vad",
				"-vad-prob-start", "85",
				"-vad-prob-continue", "70",
				"-agc",
				"-agc-level", "25000.0",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.MicFile == "test.alaw" &&
					cfg.SpeakerFile == "ref.alaw" &&
					cfg.OutputFile == "out.alaw" &&
					cfg.SampleRate == 16000 &&
					cfg.FrameSize == 320 &&
					cfg.EchoTailMs == 200 &&
					cfg.NS.NoiseSuppress == -20.0 &&
					cfg.NS.EnableVAD == true &&
					cfg.NS.VADProbStart == 85 &&
					cfg.NS.VADProbCont == 70 &&
					cfg.NS.EnableAGC == true &&
					cfg.NS.AGCLevel == 25000.0
			},
		},
		{
			name: "ns-only mode",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-ns-only",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.Mode == types.ModeNSOnly &&
					cfg.MicFile == "test.alaw"
			},
		},
		{
			name: "aec-only mode",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-speaker", "ref.alaw",
				"-aec-only",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.Mode == types.ModeAECOnly &&
					cfg.MicFile == "test.alaw" &&
					cfg.SpeakerFile == "ref.alaw"
			},
		},
		{
			name: "bypass mode",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-bypass",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.Mode == types.ModeBypass &&
					cfg.MicFile == "test.alaw"
			},
		},
		{
			name: "test-alaw mode",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-test-alaw",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.Mode == types.ModeTestAlaw &&
					cfg.MicFile == "test.alaw"
			},
		},
		{
			name: "ns-first mode",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-speaker", "ref.alaw",
				"-ns-first",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.Mode == types.ModeNSFirst &&
					cfg.MicFile == "test.alaw" &&
					cfg.SpeakerFile == "ref.alaw"
			},
		},
		{
			name: "missing mic file",
			args: []string{
				"open_tool_speex",
			},
			wantErr: true,
			check: func(cfg *types.Config) bool {
				return true // Error expected
			},
		},
		{
			name: "missing speaker file for aec mode",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-aec-only",
			},
			wantErr: true,
			check: func(cfg *types.Config) bool {
				return true // Error expected
			},
		},
		{
			name: "mutually exclusive flags",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-ns-only",
				"-aec-only",
			},
			wantErr: true,
			check: func(cfg *types.Config) bool {
				return true // Error expected
			},
		},
		{
			name: "custom filter length",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-speaker", "ref.alaw",
				"-filter-len", "4096",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.FilterLen == 4096
			},
		},
		{
			name: "prev-speaker flag",
			args: []string{
				"open_tool_speex",
				"-mic", "test.alaw",
				"-speaker", "ref.alaw",
				"-prev-speaker",
			},
			wantErr: false,
			check: func(cfg *types.Config) bool {
				return cfg.UsePrevSpeaker == true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test args
			os.Args = tt.args

			// Reset flag.CommandLine to avoid conflicts between tests
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			cfg, err := ParseFlags()

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg != nil && !tt.check(cfg) {
				t.Errorf("ParseFlags() configuration check failed")
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := types.DefaultConfig()

	// Check default values
	if cfg.OutputFile != "output.alaw" {
		t.Errorf("DefaultConfig() OutputFile = %s, want output.alaw", cfg.OutputFile)
	}

	if cfg.Mode != types.ModeAECFirst {
		t.Errorf("DefaultConfig() Mode = %v, want %v", cfg.Mode, types.ModeAECFirst)
	}

	if cfg.SampleRate != 16000 {
		t.Errorf("DefaultConfig() SampleRate = %d, want 16000", cfg.SampleRate)
	}

	if cfg.FrameSize != 320 {
		t.Errorf("DefaultConfig() FrameSize = %d, want 320", cfg.FrameSize)
	}

	if cfg.EchoTailMs != 200 {
		t.Errorf("DefaultConfig() EchoTailMs = %d, want 200", cfg.EchoTailMs)
	}

	if cfg.NS.NoiseSuppress != -15.0 {
		t.Errorf("DefaultConfig() NS.NoiseSuppress = %f, want -15.0", cfg.NS.NoiseSuppress)
	}

	if cfg.NS.EnableVAD != false {
		t.Errorf("DefaultConfig() NS.EnableVAD = %v, want false", cfg.NS.EnableVAD)
	}

	if cfg.NS.VADProbStart != 80 {
		t.Errorf("DefaultConfig() NS.VADProbStart = %d, want 80", cfg.NS.VADProbStart)
	}

	if cfg.NS.VADProbCont != 65 {
		t.Errorf("DefaultConfig() NS.VADProbCont = %d, want 65", cfg.NS.VADProbCont)
	}

	if cfg.NS.EnableAGC != false {
		t.Errorf("DefaultConfig() NS.EnableAGC = %v, want false", cfg.NS.EnableAGC)
	}

	if cfg.NS.AGCLevel != 30000.0 {
		t.Errorf("DefaultConfig() NS.AGCLevel = %f, want 30000.0", cfg.NS.AGCLevel)
	}
}
