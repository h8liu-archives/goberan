package obsim

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
	FDL
)
