package obsim

func fpAdd(x, y uint32, u, v bool) uint32 {
	xs := x >> 31
	var xe uint32
	var x0 int32

	if !u {
		xe = (x >> 23) & 0xff
		xm := ((x & 0x7fffff) << 1) | 0x1000000
		x0 = int32(xm)
		if xs != 0 {
			x0 = -x0
		}
	} else {
		xe = 150
		x0 = (int32)(x&0x00ffffff) << 8 >> 7
	}

	ys := y >> 31
	ye := (y >> 23) & 0xff
	ym := (y & 0x7fffff) << 1
	if !u && !v {
		ym |= 0x1000000
	}
	var y0 int32
	y0 = int32(ym)
	if ys != 0 {
		y0 = -y0
	}

	var e0 uint32
	var x3, y3 int32
	if ye > xe {
		shift := ye - xe
		e0 = ye
		if shift > 31 {
			x3 = x0 >> 31
		} else {
			x3 = x0 >> shift
		}
		y3 = y0
	} else {
		shift := xe - ye
		e0 = xe
		x3 = x0
		if shift > 31 {
			y3 = y0 >> 31
		} else {
			y3 = y0 >> shift
		}
	}

	sum := (xs << 26) | (xs << 25) | (uint32(x3) & 0x01ffffff)
	sum += (ys << 26) | (ys << 25) | (uint32(y3) & 0x01ffffff)

	s := sum
	if (sum & (1 << 26)) != 0 {
		s = -s
	}
	s = (s + 1) & 0x07ffffff

	e1 := e0 + 1
	t3 := s >> 1
	if (s & 0x3fffffc) != 0 {
		for (t3 & (1 << 24)) == 0 {
			t3 <<= 1
			e1--
		}
	} else {
		t3 <<= 24
		e1 -= 24
	}

	switch {
	case v:
		return uint32(int32(sum<<5) >> 6)
	case (x & 0x7fffffff) == 0:
		if u {
			return 0
		}
		return y
	case (y & 0x7fffffff) == 0:
		return x
	case (t3&0x01ffffff) == 0 || (e1&0x100) != 0:
		return 0
	default:
		return ((sum & 0x04000000) << 5) | (e1 << 23) | ((t3 >> 1) & 0x7fffff)
	}
}

func fpMul(x, y uint32) {
	panic("todo")
}

func fpDiv(x, y uint32) {
	panic("todo")
}
