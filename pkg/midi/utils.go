package midi

func decodeVarint(buf []byte) (x uint32, n int) {
	if len(buf) < 1 {
		return 0, 0
	}

	if buf[0] <= 0x80 {
		return uint32(buf[0]), 1
	}

	var b byte
	for _, b = range buf {
		x = x << 7
		x |= uint32(b) & 0x7F
		n++
		if b&0x80 == 0 {
			return x, n
		}
	}

	return x, n
}

func isVoiceMsgType(b byte) bool {
	return 0x8 <= b && b <= 0xE
}
