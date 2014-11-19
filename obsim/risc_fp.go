package obsim

import (
	"math"
)

func fpAdd(x, y uint32, u, v bool) uint32 {
	// TODO: not sure what the u and v is doing here
	fx := math.Float32frombits(x)
	fy := math.Float32frombits(y)
	return math.Float32bits(fx + fy)
}

func fpMul(x, y uint32) {
	panic("todo")
}

func fpDiv(x, y uint32) {
	panic("todo")
}
