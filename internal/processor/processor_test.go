package processor

import (
	"os"
	"path/filepath"
	"testing"

	"open_tool_speex/pkg/types"
)

func TestProcessor_Process(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()
	micFile := filepath.Join(tempDir, "mic.alaw")
	speakerFile := filepath.Join(tempDir, "speaker.alaw")
	outputFile := filepath.Join(tempDir, "output.alaw")

	// Create dummy A-law files (320 bytes each = 1 frame at 16kHz)
	createDummyAlawFile(t, micFile, 320)
	createDummyAlawFile(t, speakerFile, 320)

	tests := []struct {
		name   string
		config *types.Config
		wantErr bool
	}{
		{
			name: "bypass mode",
			config: &types.Config{
				MicFile:     micFile,
				SpeakerFile: speakerFile,
				OutputFile:  outputFile,
				Mode:        types.ModeBypass,
				SampleRate:  16000,
				FrameSize:   320,
			},
			wantErr: false,
		},
		{
			name: "test alaw mode",
			config: &types.Config{
				MicFile:     micFile,
				SpeakerFile: speakerFile,
				OutputFile:  outputFile,
				Mode:        types.ModeTestAlaw,
				SampleRate:  16000,
				FrameSize:   320,
			},
			wantErr: false,
		},
		{
			name: "ns only mode",
			config: &types.Config{
				MicFile:     micFile,
				SpeakerFile: speakerFile,
				OutputFile:  outputFile,
				Mode:        types.ModeNSOnly,
				SampleRate:  16000,
				FrameSize:   320,
				NS: types.NSConfig{
					NoiseSuppress: -15.0,
					EnableVAD:     false,
					VADProbStart:  80,
					VADProbCont:   65,
					EnableAGC:     false,
					AGCLevel:      30000.0,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(tt.config)
			err := processor.Process()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Processor.Process() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check if output file was created
			if !tt.wantErr {
				if _, err := os.Stat(tt.config.OutputFile); os.IsNotExist(err) {
					t.Errorf("Output file was not created: %s", tt.config.OutputFile)
				}
			}
		})
	}
}

func TestProcessor_needsSpeakerFile(t *testing.T) {
	tests := []struct {
		name   string
		config *types.Config
		want   bool
	}{
		{
			name: "bypass mode",
			config: &types.Config{Mode: types.ModeBypass},
			want:  false,
		},
		{
			name: "test alaw mode",
			config: &types.Config{Mode: types.ModeTestAlaw},
			want:  false,
		},
		{
			name: "ns only mode",
			config: &types.Config{Mode: types.ModeNSOnly},
			want:  false,
		},
		{
			name: "aec only mode",
			config: &types.Config{Mode: types.ModeAECOnly},
			want:  true,
		},
		{
			name: "ns first mode",
			config: &types.Config{Mode: types.ModeNSFirst},
			want:  true,
		},
		{
			name: "aec first mode",
			config: &types.Config{Mode: types.ModeAECFirst},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := &Processor{config: tt.config}
			if got := processor.needsSpeakerFile(); got != tt.want {
				t.Errorf("Processor.needsSpeakerFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessor_getAECSpeakerFrame(t *testing.T) {
	speakerFrame := []int16{1, 2, 3, 4, 5}
	prevSpeakerFrame := []int16{10, 20, 30, 40, 50}

	tests := []struct {
		name           string
		config         *types.Config
		speakerFrame   []int16
		prevSpeakerFrame []int16
		want           []int16
	}{
		{
			name: "use current speaker frame",
			config: &types.Config{UsePrevSpeaker: false},
			speakerFrame: speakerFrame,
			prevSpeakerFrame: prevSpeakerFrame,
			want: speakerFrame,
		},
		{
			name: "use previous speaker frame",
			config: &types.Config{UsePrevSpeaker: true},
			speakerFrame: speakerFrame,
			prevSpeakerFrame: prevSpeakerFrame,
			want: prevSpeakerFrame,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := &Processor{config: tt.config}
			got := processor.getAECSpeakerFrame(tt.speakerFrame, tt.prevSpeakerFrame)
			
			if len(got) != len(tt.want) {
				t.Errorf("Processor.getAECSpeakerFrame() length = %d, want %d", len(got), len(tt.want))
				return
			}
			
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Processor.getAECSpeakerFrame()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Helper function to create dummy A-law files
func createDummyAlawFile(t *testing.T, filename string, size int) {
	t.Helper()
	
	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", filename, err)
	}
	defer file.Close()

	// Write A-law silence (0xD5) bytes
	data := make([]byte, size)
	for i := range data {
		data[i] = 0xD5
	}
	
	if _, err := file.Write(data); err != nil {
		t.Fatalf("Failed to write test data to %s: %v", filename, err)
	}
}
