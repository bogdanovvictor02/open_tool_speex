package main

// A-law encoding/decoding for 16 kHz mono audio
// ITU-T G.711 A-law companding

var alawDecodeTable = [256]int16{
	-5504, -5248, -6016, -5760, -4480, -4224, -4992, -4736,
	-7552, -7296, -8064, -7808, -6528, -6272, -7040, -6784,
	-2752, -2624, -3008, -2880, -2240, -2112, -2496, -2368,
	-3776, -3648, -4032, -3904, -3264, -3136, -3520, -3392,
	-22016, -20992, -24064, -23040, -17920, -16896, -19968, -18944,
	-30208, -29184, -32256, -31232, -26112, -25088, -28160, -27136,
	-11008, -10496, -12032, -11520, -8960, -8448, -9984, -9472,
	-15104, -14592, -16128, -15616, -13056, -12544, -14080, -13568,
	-344, -328, -376, -360, -280, -264, -312, -296,
	-472, -456, -504, -488, -408, -392, -440, -424,
	-88, -72, -120, -104, -24, -8, -56, -40,
	-216, -200, -248, -232, -152, -136, -184, -168,
	-1376, -1312, -1504, -1440, -1120, -1056, -1248, -1184,
	-1888, -1824, -2016, -1952, -1632, -1568, -1760, -1696,
	-688, -656, -752, -720, -560, -528, -624, -592,
	-944, -912, -1008, -976, -816, -784, -880, -848,
	5504, 5248, 6016, 5760, 4480, 4224, 4992, 4736,
	7552, 7296, 8064, 7808, 6528, 6272, 7040, 6784,
	2752, 2624, 3008, 2880, 2240, 2112, 2496, 2368,
	3776, 3648, 4032, 3904, 3264, 3136, 3520, 3392,
	22016, 20992, 24064, 23040, 17920, 16896, 19968, 18944,
	30208, 29184, 32256, 31232, 26112, 25088, 28160, 27136,
	11008, 10496, 12032, 11520, 8960, 8448, 9984, 9472,
	15104, 14592, 16128, 15616, 13056, 12544, 14080, 13568,
	344, 328, 376, 360, 280, 264, 312, 296,
	472, 456, 504, 488, 408, 392, 440, 424,
	88, 72, 120, 104, 24, 8, 56, 40,
	216, 200, 248, 232, 152, 136, 184, 168,
	1376, 1312, 1504, 1440, 1120, 1056, 1248, 1184,
	1888, 1824, 2016, 1952, 1632, 1568, 1760, 1696,
	688, 656, 752, 720, 560, 528, 624, 592,
	944, 912, 1008, 976, 816, 784, 880, 848,
}

var alawEncodeTable [4096]uint8

func init() {
	// Build A-law encoding table on startup
	for i := 0; i < 4096; i++ {
		val := int16(i)
		alawEncodeTable[i] = pcm16ToAlawDirect(val)
	}
}

// AlawToPCM16 converts A-law sample to 16-bit PCM
func AlawToPCM16(alaw uint8) int16 {
	return alawDecodeTable[alaw]
}

// PCM16ToAlaw converts 16-bit PCM sample to A-law
func PCM16ToAlaw(pcm int16) uint8 {
	if pcm >= 0 {
		if pcm >= 4096 {
			pcm = 4095
		}
		return alawEncodeTable[pcm]
	} else {
		pcm = -pcm - 1
		if pcm >= 4096 {
			pcm = 4095
		}
		return alawEncodeTable[pcm] ^ 0x80
	}
}

// pcm16ToAlawDirect performs direct A-law encoding (used in table generation)
func pcm16ToAlawDirect(pcm int16) uint8 {
	var mask uint8
	var seg uint8

	if pcm >= 0 {
		pcm = pcm >> 4
	} else {
		mask = 0x80
		pcm = (-pcm - 1) >> 4
		if pcm < 0 {
			pcm = 0x7ff
		}
	}

	if pcm >= 256 {
		seg = 8
		for i := 0x100; i != 0; i >>= 1 {
			seg--
			if (pcm & int16(i)) != 0 {
				break
			}
		}
	} else {
		seg = 1
		for i := int16(0x20); i != 0; i >>= 1 {
			seg++
			if (pcm & i) != 0 {
				break
			}
		}
	}

	if seg >= 8 {
		return 0x7f ^ mask
	}

	alaw := uint8((seg << 4) | uint8((pcm>>(seg+3))&0x0f))
	return alaw ^ 0x55 ^ mask
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
