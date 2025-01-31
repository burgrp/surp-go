package surp

// Calculates SURP hash for the given string, which is the CRC16-CCITT.
func CalculateHash(name string) uint16 {
	var crc uint16 = 0xFFFF
	for _, c := range name {
		crc ^= uint16(c) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
