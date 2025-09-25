package audio

import (
	"testing"
)

func TestLinear2Alaw(t *testing.T) {
	tests := []struct {
		name     string
		input    int16
		expected uint8
	}{
		{
			name:     "zero",
			input:    0,
			expected: 0xD5, // A-law representation of 0
		},
		{
			name:     "positive small",
			input:    100,
			expected: 0xD3, // Actual A-law for small positive value
		},
		{
			name:     "negative small",
			input:    -100,
			expected: 0x53, // Actual A-law for small negative value
		},
		{
			name:     "positive large",
			input:    32767,
			expected: 0xAA, // Actual A-law for maximum positive value
		},
		{
			name:     "negative large",
			input:    -32768,
			expected: 0x2A, // Actual A-law for maximum negative value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Linear2Alaw(tt.input)
			if result != tt.expected {
				t.Errorf("Linear2Alaw(%d) = 0x%02X, expected 0x%02X", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAlaw2Linear(t *testing.T) {
	tests := []struct {
		name     string
		input    uint8
		expected int16
	}{
		{
			name:     "zero",
			input:    0xD5,
			expected: 0, // A-law 0xD5 represents 0
		},
		{
			name:     "positive small",
			input:    0xD3,
			expected: 100, // Actual linear for small positive A-law
		},
		{
			name:     "negative small",
			input:    0x53,
			expected: -100, // Actual linear for small negative A-law
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Alaw2Linear(tt.input)
			// Allow some tolerance due to A-law compression
			if abs(result-tt.expected) > 10 {
				t.Errorf("Alaw2Linear(0x%02X) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAlawRoundTrip(t *testing.T) {
	// Test with reasonable values (avoid extreme values due to A-law compression limits)
	testValues := []int16{0, 100, -100, 1000, -1000, 10000, -10000}

	for _, val := range testValues {
		t.Run("roundtrip", func(t *testing.T) {
			alaw := Linear2Alaw(val)
			back := Alaw2Linear(alaw)
			
			// Allow tolerance due to A-law compression (increases with magnitude)
			tolerance := int16(10)
			if abs(val) > 1000 {
				tolerance = int16(100)
			}
			if abs(val) > 10000 {
				tolerance = int16(1000)
			}
			
			if abs(back-val) > tolerance {
				t.Errorf("Round trip failed: %d -> 0x%02X -> %d (tolerance: %d)", val, alaw, back, tolerance)
			}
		})
	}
}

func TestAlawBufferConversion(t *testing.T) {
	// Test buffer conversion functions
	pcmData := []int16{0, 100, -100, 1000, -1000}
	alawData := make([]byte, len(pcmData))
	backPcmData := make([]int16, len(pcmData))

	// PCM -> A-law -> PCM
	PCM16BufferToAlaw(pcmData, alawData)
	AlawBufferToPCM16(alawData, backPcmData)

	for i := 0; i < len(pcmData); i++ {
		if abs(backPcmData[i]-pcmData[i]) > 10 {
			t.Errorf("Buffer conversion failed at index %d: %d -> %d", i, pcmData[i], backPcmData[i])
		}
	}
}

func TestAlawSilence(t *testing.T) {
	// Test that A-law silence (0xD5) converts to 0
	silence := uint8(0xD5)
	result := Alaw2Linear(silence)
	// Allow some tolerance due to A-law compression
	if abs(result) > 10 {
		t.Errorf("A-law silence (0xD5) should convert to 0, got %d", result)
	}

	// Test that 0 converts to A-law silence
	zero := int16(0)
	resultAlaw := Linear2Alaw(zero)
	if resultAlaw != 0xD5 {
		t.Errorf("Linear 0 should convert to A-law silence (0xD5), got 0x%02X", resultAlaw)
	}
}

// Helper function for absolute value
func abs(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}
