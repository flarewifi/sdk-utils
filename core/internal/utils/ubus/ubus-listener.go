package ubus

type UbusListener struct {
	outCh chan []byte
}

func (self *UbusListener) Write(b []byte) (n int, err error) {
	self.outCh <- b
	return len(b), nil
}

func (self *UbusListener) OutCh() <-chan []byte {
	return self.outCh
}

func NewUbusListener() *UbusListener {
	return &UbusListener{
		outCh: make(chan []byte),
	}
}
