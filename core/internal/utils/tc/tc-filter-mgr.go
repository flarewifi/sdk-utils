package tc

type TcFilterMgr struct {
	dev      string
	tcFilter *TcFilter
}

func NewTcFilterMgr(dev string) *TcFilterMgr {
	return &TcFilterMgr{
		dev: dev,
	}
}

func (self *TcFilterMgr) Setup(ip string, netmask int) (err error) {
	filter, err := NewTcFilter(self.dev, ip, netmask)
	if err != nil {
		return err
	}
	if err := filter.Setup(); err != nil {
		return err
	}

	self.tcFilter = filter

	return nil
}

func (self *TcFilterMgr) Reset() (err error) {
	return self.tcFilter.Reset()
}

func (self *TcFilterMgr) CreateFilter(clientIp string, classid TcClassId) error {
	return self.tcFilter.CreateFilter(clientIp, classid.String())
}

func (self *TcFilterMgr) DeleteFilter(clientIp string) error {
	return self.tcFilter.DeleteFilter(clientIp)
}

func (self *TcFilterMgr) CleanUp() error {
	return self.tcFilter.CleanUp()
}
