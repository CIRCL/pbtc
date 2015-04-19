package util

func MinUint32(x uint32, y uint32) uint32 {
	if x > y {
		return x
	}

	return y
}
