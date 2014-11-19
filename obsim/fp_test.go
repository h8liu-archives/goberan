package obsim

import (
	"testing"

	"math"
	"math/rand"
)

func TestFp(t *testing.T) {
	var nums []uint32

	for e := uint32(0); e < 256; e++ {
		nums = append(nums, e<<23)
		nums = append(nums, (e<<23)|1)
		nums = append(nums, (e<<23)|0x7fffff)
		nums = append(nums, (e<<23)|0x7ffffe)
	}

	var nums2 []uint32
	for _, x := range nums {
		nums2 = append(nums, x|0x80000000)
	}
	nums = append(nums, nums2...)

	for i := 0; i < 1000; i++ {
		nums = append(nums, rand.Uint32())
	}

	nerr := 0
	for _, n1 := range nums {
		f1 := math.Float32frombits(n1)
		for _, n2 := range nums {
			f2 := math.Float32frombits(n2)
			fexpect := f1 + f2
			expect := math.Float32bits(fexpect)
			got := fpAdd(n1, n2, false, false)
			fgot := math.Float32frombits(got)

			if got != expect {
				t.Errorf("%08x(%f) + %08x(%f) = %08x(%f), but got %08x(%f)",
					n1, f1,
					n2, f2,
					expect, fexpect,
					got, fgot,
				)
				nerr++
				if nerr > 10 {
					t.FailNow()
				}
			}
		}
	}
}
