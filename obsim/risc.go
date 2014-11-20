package obsim

import (
	"fmt"
	"os"
)

const (
	RISCFrameBufferWidth  = 1024
	RISCFrameBufferHeight = 768

	MemSize  = 0x00180000
	MemWords = (MemSize / 4)
	ROMStart = 0xfffff800
	ROMWords = 512

	DisplayStart = 0x000e7f00
	IOStart      = 0xffffffc0
)

type Damage struct {
	x1, x2, y1, y2 int
}

type RISC struct {
	PC         uint32
	R          []uint32
	H          uint32
	Z, N, C, V bool

	progress    uint32
	currentTick uint32
	mouse       uint32
	keyBuf      []uint8
	keyCnt      uint32
	leds        uint32

	serial      RiscSerial
	spiSelected uint32

	spi       []RiscSPI
	clipboard RiscClipboard

	fbWidth  int
	fbHeight int
	damage   *Damage

	RAM []uint32
	ROM []uint32
}

func NewRISC() *RISC {
	ret := new(RISC)
	ret.R = make([]uint32, 16)
	ret.keyBuf = make([]uint8, 16)
	ret.spi = make([]RiscSPI, 4)
	ret.RAM = make([]uint32, MemWords)
	ret.ROM = make([]uint32, ROMWords)

	copy(ret.ROM, bootloader)
	ret.screenSizeHack(RISCFrameBufferWidth, RISCFrameBufferHeight)

	ret.reset()

	return ret
}

func (r *RISC) screenSizeHack(width, height int) {
	r.fbWidth = width / 32
	r.fbHeight = height
	r.damage = &Damage{
		x1: 0,
		y1: 0,
		x2: r.fbWidth - 1,
		y2: r.fbHeight - 1,
	}

	r.RAM[DisplayStart/4] = 0x53697a66
	r.RAM[DisplayStart/4+1] = uint32(width)
	r.RAM[DisplayStart/4+2] = uint32(height)
}

func (r *RISC) reset() {
	r.PC = ROMStart / 4
}

func (r *RISC) setSerial(serial RiscSerial) {
	r.serial = serial
}

func (r *RISC) setSPI(index int, spi RiscSPI) {
	if index == 1 || index == 2 {
		r.spi[index] = spi
	}
}

func (r *RISC) setClipboard(clipboard RiscClipboard) {
	r.clipboard = clipboard
}

func (r *RISC) run(cycles int) {
	r.progress = 20

	for i := 0; i < cycles && r.progress != 0; i++ {
		r.singleStep()
	}
}

func (r *RISC) singleStep() {
	var ir uint32
	if r.PC < MemWords {
		ir = r.RAM[r.PC]
	} else if r.PC >= ROMStart/4 && r.PC < ROMStart/4+ROMWords {
		ir = r.ROM[r.PC-ROMStart/4]
	} else {
		fmt.Fprintf(os.Stderr, "Branched into the void (PC=0x%08x), resetting...\n", r.PC)
		r.reset()
		return
	}

	r.PC++

	const (
		pbit uint32 = 0x80000000
		qbit uint32 = 0x40000000
		ubit uint32 = 0x20000000
		vbit uint32 = 0x10000000
	)

	if (ir & pbit) == 0 {
		a := (ir & 0x0f000000) >> 24
		b := (ir & 0x00f00000) >> 20
		op := (ir & 0x000f0000) >> 16
		im := ir & 0x0000ffff
		c := ir & 0x0000000f

		var aVal, bVal, cVal uint32

		bVal = r.R[b]
		if (ir & qbit) == 0 {
			cVal = r.R[c]
		} else if (ir & vbit) == 0 {
			cVal = im
		} else {
			cVal = 0xffff0000 | im
		}

		switch op {
		case MOV:
			if (ir & ubit) == 0 {
				aVal = cVal
			} else if (ir & qbit) != 0 {
				aVal = cVal << 16
			} else if (ir & vbit) != 0 {
				aVal = 0xd0
				if r.N {
					aVal |= 0x80000000
				}
				if r.Z {
					aVal |= 0x40000000
				}
				if r.C {
					aVal |= 0x20000000
				}
				if r.V {
					aVal |= 0x10000000
				}
			} else {
				aVal = r.H
			}

		case LSL:
			aVal = bVal << (cVal & 31)
		case ASR:
			aVal = uint32((int32)(bVal) >> (cVal & 31))
		case ROR:
			aVal = bVal >> (cVal & 31)
			aVal |= bVal << (-cVal & 31)
		case AND:
			aVal = bVal & cVal
		case ANN:
			aVal = bVal & ^cVal
		case IOR:
			aVal = bVal | cVal
		case XOR:
			aVal = bVal ^ cVal
		case ADD:
			aVal = bVal + cVal
			if (ir&ubit) != 0 && r.C {
				aVal++
			}

			r.C = aVal < bVal
			r.V = ((aVal ^ bVal) & (aVal ^ bVal) >> 31) != 0

		case SUB:
			aVal = bVal - cVal
			if (ir&ubit) != 0 && r.C {
				aVal--
			}

			r.C = aVal > bVal
			r.V = (((bVal ^ cVal) & (aVal ^ bVal)) >> 31) != 0

		case MUL:
			var tmp uint64
			if (ir & ubit) == 0 {
				tmp = uint64(int64(int32(bVal)) * int64(int32(cVal)))
			} else {
				tmp = uint64(bVal) * uint64(cVal)
			}

			aVal = uint32(tmp)
			r.H = uint32(tmp >> 32)

		case DIV:
			if int32(cVal) <= 0 {
				fmt.Fprintf(os.Stderr,
					"ERROR: PC 0x%08x: divisor %d is not positive\n",
					r.PC*4-4,
					cVal,
				)
			} else {
				aVal = uint32(int32(bVal) / int32(cVal))
				r.H = uint32(int32(bVal) % int32(cVal))
				if int32(r.H) < 0 {
					aVal--
					r.H += cVal
				}
			}

		case FAD:
			aVal = fpAdd(bVal, cVal, (ir&ubit) != 0, (ir&vbit) != 0)

		case FSB:
			aVal = fpAdd(bVal, cVal^0x80000000,
				(ir&ubit) != 0, (ir&vbit) != 0)

		case FML:
			aVal = fpMul(bVal, cVal)

		case FDV:
			aVal = fpDiv(bVal, cVal)

		default:
			panic("abort")
		}

		r.setRegister(int(a), aVal)
	} else if (ir & qbit) == 0 {
		// memory  instructions
		a := (ir & 0x0f000000) >> 24
		b := (ir & 0x00f00000) >> 20
		off := ir & 0x000fffff
		off = (off ^ 0x00080000) - 0x00080000 // sign-extend

		address := r.R[b] + off
		if (ir & ubit) == 0 {
			var aVal uint32

			if (ir & vbit) == 0 {
				aVal = r.loadWord(address)
			} else {
				aVal = uint32(r.loadByte(address))
			}
			r.setRegister(int(a), aVal)
		} else {
			if (ir & vbit) == 0 {
				r.storeWord(address, r.R[a])
			} else {
				r.storeByte(address, uint8(r.R[a]))
			}
		}
	} else {
		// branch instructions
		var t bool
		switch (ir >> 24) & 7 {
		case 0:
			t = r.N
		case 1:
			t = r.Z
		case 2:
			t = r.C
		case 3:
			t = r.V
		case 4:
			t = r.C || r.Z
		case 5:
			t = r.N != r.V
		case 6:
			t = (r.N != r.V) || r.Z
		case 7:
			t = true
		default:
			panic("bug")
		}

		if (ir >> 27) & 0x1 != 0 {
			t = !t
		}

		if t {
			if (ir & vbit) != 0 {
				r.setRegister(15, r.PC * 4)
			}

			if (ir & ubit) == 0 {
				c := ir & 0x0000000f
				r.PC = r.R[c] / 4
			} else {
				off := ir & 0x00ffffff
				off = (off & 0x00800000) - 0x00800000 // sign-extend
				r.PC = r.PC + off
			}
		}
	}
}

func (r *RISC) setRegister(reg int, value uint32) {
	r.R[reg] = value
	r.Z = (value == 0)
	r.N = int32(value) < 0
}

func (r *RISC) loadWord(address uint32) uint32 {
	if address < MemSize {
		return r.RAM[address/4]
	} else {
		return r.loadIO(address)
	}
}

func (r *RISC) loadByte(address uint32) uint8 {
	w := r.loadWord(address)
	return uint8(w >> (address % 4 * 8))
}

func (r *RISC) storeByte(address uint32, value uint8) {
	if address < MemSize {
		w := r.loadWord(address)
		shift := (address & 3) * 8
		w &= ^(uint32(0xff) << shift)
		w |= uint32(value) << shift
		r.storeWord(address, w)
	} else {
		r.storeIO(address, uint32(value))
	}
}

func (r *RISC) storeWord(address uint32, value uint32) {
	if address < DisplayStart {
		r.RAM[address/4] = value
	} else if address < MemSize {
		r.RAM[address/4] = value
		r.updateDamage(int(address/4 - DisplayStart/4))
	} else {
		r.storeIO(address, value)
	}
}

func (r *RISC) updateDamage(w int) {
	row := w / r.fbWidth
	col := w % r.fbWidth
	if row >= r.fbHeight {
		return
	}

	if col < r.damage.x1 {
		r.damage.x1 = col
	}
	if col < r.damage.x2 {
		r.damage.x2 = col
	}
	if row < r.damage.y1 {
		r.damage.y1 = row
	}
	if row < r.damage.y2 {
		r.damage.y2 = row
	}
}

func (r *RISC) loadIO(address uint32) uint32 {
	panic("todo")
}

func (r *RISC) storeIO(address uint32, value uint32) {
	panic("todo")
}

const (
	MOV = iota
	LSL
	ASR
	ROR
	AND
	ANN
	IOR
	XOR
	ADD
	SUB
	MUL
	DIV
	FAD
	FSB
	FML
	FDV
)
