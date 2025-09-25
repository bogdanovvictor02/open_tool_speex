package main

// A-law encoding/decoding for 16 kHz mono audio
// ITU-T G.711 A-law companding
// Implementation based on CCITT G.711 specifications

const (
	SIGN_BIT   = 0x80 // Sign bit for a A-law byte
	QUANT_MASK = 0xf  // Quantization field mask
	NSEGS      = 8    // Number of A-law segments
	SEG_SHIFT  = 4    // Left shift for segment number
	SEG_MASK   = 0x70 // Segment field mask
)

var segAend = [8]int{0x1F, 0x3F, 0x7F, 0xFF, 0x1FF, 0x3FF, 0x7FF, 0xFFF}
var segUend = [8]int{0x3F, 0x7F, 0xFF, 0x1FF, 0x3FF, 0x7FF, 0xFFF, 0x1FFF}

// search finds the segment for a given value
func search(val int, table []int, size int) int {
	for i := 0; i < size; i++ {
		if val <= table[i] {
			return i
		}
	}
	return size
}

// Linear2Alaw converts a 16-bit linear PCM value to 8-bit A-law
// Based on CCITT G.711 specifications
func Linear2Alaw(pcmVal int16) uint8 {
	mask := uint8(0)
	seg := 0
	aval := 0

	pcmVal = pcmVal >> 3

	if pcmVal >= 0 {
		mask = 0xD5 // sign (7th) bit = 1
	} else {
		mask = 0x55 // sign bit = 0
		pcmVal = -pcmVal - 1
	}

	// Convert the scaled magnitude to segment number
	seg = search(int(pcmVal), segAend[:], 8)

	// Combine the sign, segment, and quantization bits
	if seg >= 8 { // out of range, return maximum value
		return 0x7F ^ mask
	} else {
		aval = seg << SEG_SHIFT
		if seg < 2 {
			aval |= (int(pcmVal) >> 1) & QUANT_MASK
		} else {
			aval |= (int(pcmVal) >> seg) & QUANT_MASK
		}
		return uint8(aval ^ int(mask))
	}
}

// Alaw2Linear converts an A-law value to 16-bit linear PCM
// Based on CCITT G.711 specifications
func Alaw2Linear(aVal uint8) int16 {
	t := 0
	seg := 0

	aVal ^= 0x55

	t = (int(aVal) & QUANT_MASK) << 4
	seg = (int(aVal) & SEG_MASK) >> SEG_SHIFT

	switch seg {
	case 0:
		t += 8
	case 1:
		t += 0x108
	default:
		t += 0x108
		t <<= seg - 1
	}

	if (aVal & SIGN_BIT) != 0 {
		return int16(t)
	} else {
		return int16(-t)
	}
}

// AlawToPCM16 converts A-law sample to 16-bit PCM
func AlawToPCM16(alaw uint8) int16 {
	return Alaw2Linear(alaw)
}

// PCM16ToAlaw converts 16-bit PCM sample to A-law
func PCM16ToAlaw(pcm int16) uint8 {
	return Linear2Alaw(pcm)
}

// AlawBufferToPCM16 converts A-law buffer to PCM16 buffer
func AlawBufferToPCM16(alawData []byte, pcmData []int16) {
	for i := 0; i < len(alawData) && i < len(pcmData); i++ {
		pcmData[i] = AlawToPCM16(alawData[i])
	}
}

// PCM16BufferToAlaw converts PCM16 buffer to A-law buffer
func PCM16BufferToAlaw(pcmData []int16, alawData []byte) {
	for i := 0; i < len(pcmData) && i < len(alawData); i++ {
		alawData[i] = PCM16ToAlaw(pcmData[i])
	}
}
