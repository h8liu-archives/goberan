package obsim

type RiscSerial interface {
	ReadStatus() uint32
	ReadData() uint32
	WriteData(dat uint32)
}

type RiscSPI interface {
	ReadData() uint32
	WriteData(dat uint32)
}

type RiscClipboard interface {
	WriteControl(d uint32)
	ReadControl() uint32
	WriteData(d uint32)
	ReadData() uint32
}
