package tc

import (
	"fmt"

	ifbutil "core/internal/modules/network"
	jobque "core/tools/job-que"
	cmd "core/tools/shell"
)

var tcClassQue = jobque.NewJobQue[any]()

type TcClassMgr struct {
	dev       string
	download  Kbit
	upload    Kbit
	classList []*TcClass
}

func NewTcClassMgr(dev string, download Kbit, upload Kbit) *TcClassMgr {
	return &TcClassMgr{
		dev:       dev,
		download:  download,
		upload:    upload,
		classList: []*TcClass{},
	}
}

func (self *TcClassMgr) Bandwidth() (download Kbit, upload Kbit) {
	d := self.download
	u := self.upload
	return d, u
}

func (self *TcClassMgr) Setup() error {
	_, err := tcClassQue.Exec(func() (any, error) {
		var err error
		dev := self.dev
		ifb := ifbName(dev)
		rootId := TcClassIdRoot.String()
		defid := TcClassIdDefault

		// Clean up old TC rules (ignore errors - may not exist on first setup)
		self.CleanUp()

		calls := []string{}
		if ifbutil.IsIfbSupported() {
			calls = []string{
				fmt.Sprintf("ip link add name %s type ifb", ifb),
				fmt.Sprintf("ip link set dev %s up", ifb),
				fmt.Sprintf("tc qdisc add dev %s handle ffff: ingress", dev),
				fmt.Sprintf("tc filter add dev %s parent ffff: protocol ip u32 match u32 0 0 action mirred egress redirect dev %s", dev, ifb),
				fmt.Sprintf("tc qdisc add dev %s root handle %s hfsc default %d", dev, rootId, defid),
				fmt.Sprintf("tc qdisc add dev %s root handle %s hfsc default %d", ifb, rootId, defid),
			}
		} else {
			calls = []string{
				fmt.Sprintf("tc qdisc add dev %s handle ffff: ingress", dev),
				fmt.Sprintf("tc qdisc add dev %s root handle %s hfsc default %d", dev, rootId, defid),
			}
		}

		for _, c := range calls {
			err := cmd.Exec(c, nil)
			if err != nil {
				return nil, err
			}
		}

		err = self.tcAddOrChange(tcActionAdd, TcClassIdRoot, self.DefaultTcClass())
		if err != nil {
			return nil, err
		}

		err = self.tcAddOrChange(tcActionAdd, TcClassIdRoot, self.UserTcClass())
		if err != nil {
			return nil, err
		}

		return nil, nil
	})

	return err
}

func (self *TcClassMgr) Reset() (err error) {
	err = self.Setup()
	if err != nil {
		return err
	}

	_, err = tcClassQue.Exec(func() (any, error) {
		for _, c := range self.classList {
			err = self.tcAddOrChange(tcActionAdd, TcClassIdUser, c)
			if err != nil {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}

func (self *TcClassMgr) UpdateBandwidth(down Kbit, up Kbit) error {
	_, err := tcClassQue.Exec(func() (any, error) {
		self.download = down
		self.upload = up
		err := self.tcAddOrChange(tcActionChange, TcClassIdRoot, self.DefaultTcClass())
		if err != nil {
			return nil, err
		}

		err = self.tcAddOrChange(tcActionChange, TcClassIdRoot, self.UserTcClass())
		if err != nil {
			return nil, err
		}

		return nil, nil
	})
	return err
}

func (self *TcClassMgr) CreateClass(parent *TcClass, classid TcClassId, down Kbit, up Kbit, ceilDown Kbit, ceilUp Kbit) error {
	_, err := tcClassQue.Exec(func() (any, error) {
		klass := NewTcClass(parent, classid, down, up, ceilDown, ceilUp)
		klass.Sanitize()

		err := self.tcAddOrChange(tcActionAdd, parent.ClassId, klass)
		if err != nil {
			return nil, err
		}

		self.classList = append(self.classList, klass)
		return nil, nil

	})

	return err
}

func (self *TcClassMgr) ChangeClass(parent *TcClass, classid TcClassId, down Kbit, up Kbit, ceilDown Kbit, ceilUp Kbit) error {
	_, err := tcClassQue.Exec(func() (any, error) {
		for _, klass := range self.classList {
			if klass.ClassId == classid {
				klass.MinDown = down
				klass.MinUp = up
				klass.CeilDown = ceilDown
				klass.CeilUp = ceilUp
				klass.Sanitize()

				return nil, self.tcAddOrChange(tcActionChange, parent.ClassId, klass)
			}
		}
		return nil, fmt.Errorf("classid %d does not exist on interface %s", int(classid), self.dev)
	})

	return err
}

func (self *TcClassMgr) DeleteClass(classid TcClassId) error {
	_, err := tcClassQue.Exec(func() (any, error) {
		for i, klass := range self.classList {
			if klass.ClassId == classid {
				err := self.tcDel(klass)
				self.classList = append(self.classList[:i], self.classList[i+1:]...)
				return nil, err
			}
		}
		return nil, fmt.Errorf("classid %d does not exist on interface %s", int(classid), self.dev)
	})

	return err
}

func (self *TcClassMgr) tcAddOrChange(action tcAction, parent TcClassId, klass *TcClass) (err error) {
	ifb := ifbName(self.dev)
	classid := klass.ClassId.String()
	parentid := parent.String()

	if err = cmd.Exec(fmt.Sprintf(`tc class %s dev %s parent %s classid %s hfsc ls rate %dkbit ul rate %dkbit`, action, self.dev, parentid, classid, klass.MinDown, klass.CeilDown), nil); err != nil {
		return err
	}

	if ifbutil.IsIfbSupported() {
		if err = cmd.Exec(fmt.Sprintf(`tc class %s dev %s parent %s classid %s hfsc ls rate %dkbit ul rate %dkbit`, action, ifb, parentid, classid, klass.MinUp, klass.CeilUp), nil); err != nil {
			return err
		}
	}

	return nil
}

func (self *TcClassMgr) tcDel(klass *TcClass) error {
	classid := klass.ClassId.String()
	ifb := ifbName(self.dev)

	if err := cmd.Exec(fmt.Sprintf("tc class del dev %s classid %s", self.dev, classid), nil); err != nil {
		return err
	}

	if ifbutil.IsIfbSupported() {
		if err := cmd.Exec(fmt.Sprintf("tc class del dev %s classid %s", ifb, classid), nil); err != nil {
			return err
		}
	}

	klass.ClassId.Restore()

	return nil
}

func (self *TcClassMgr) RootTcClass() *TcClass {
	return &TcClass{
		ClassId:  TcClassIdRoot,
		MinDown:  self.download,
		MinUp:    self.upload,
		CeilDown: self.download,
		CeilUp:   self.upload,
	}
}

func (self *TcClassMgr) DefaultTcClass() *TcClass {
	minDown := self.download / 2
	minUp := self.upload / 2
	return NewTcClass(self.RootTcClass(), TcClassIdDefault, minDown, minUp, self.download, self.upload)
}

func (self *TcClassMgr) UserTcClass() *TcClass {
	minDown := self.download / 2
	minUp := self.upload / 2
	return NewTcClass(self.RootTcClass(), TcClassIdUser, minDown, minUp, self.download, self.upload)
}

func (self *TcClassMgr) CleanUp() error {
	dev := self.dev
	ifb := ifbName(dev)

	cmd.Exec(fmt.Sprintf("tc qdisc del dev %s root", dev), nil)
	cmd.Exec(fmt.Sprintf("tc qdisc del dev %s ingress", dev), nil)
	cmd.Exec(fmt.Sprintf("tc qdisc del dev %s root", ifb), nil) // this migt not be necessary since we're deleting the interface
	cmd.Exec(fmt.Sprintf("ip link delete %s", ifb), nil)

	return nil
}
