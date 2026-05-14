package tc

import (
	"fmt"
	"log"
	"sync"

	ifbutil "core/internal/modules/network"
	jobque "core/utils/job-que"
	cmd "core/utils/shell"
)

var tcClassQue = jobque.NewJobQueue[any]()

type TcClassMgr struct {
	mu        sync.RWMutex
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
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.download, self.upload
}

func (self *TcClassMgr) Setup() error {
	_, err := tcClassQue.Exec("TcClassMgr.Setup", func() (any, error) {
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
				fmt.Sprintf("tc qdisc add dev %s root handle %s htb default %d", dev, rootId, defid),
				fmt.Sprintf("tc qdisc add dev %s root handle %s htb default %d", ifb, rootId, defid),
			}
		} else {
			calls = []string{
				fmt.Sprintf("tc qdisc add dev %s handle ffff: ingress", dev),
				fmt.Sprintf("tc qdisc add dev %s root handle %s htb default %d", dev, rootId, defid),
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

		// Add fq_codel leaf qdisc for bufferbloat protection on user class only.
		// Default class 1:1 intentionally left with HTB's inherited pfifo so
		// system/control-plane traffic isn't subject to flow-isolation queuing.
		if err = self.addLeafQdisc(TcClassIdUser); err != nil {
			log.Printf("Warning: Failed to add fq_codel to user class: %v", err)
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

	_, err = tcClassQue.Exec("TcClassMgr.Reset.restoreClasses", func() (any, error) {
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
	_, err := tcClassQue.Exec("TcClassMgr.UpdateBandwidth", func() (any, error) {
		self.mu.Lock()
		self.download = down
		self.upload = up
		self.mu.Unlock()

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
	_, err := tcClassQue.Exec("TcClassMgr.CreateClass", func() (any, error) {
		klass := NewTcClass(parent, classid, down, up, ceilDown, ceilUp)
		klass.Sanitize()

		err := self.tcAddOrChange(tcActionAdd, parent.ClassId, klass)
		if err != nil {
			return nil, err
		}

		// Add fq_codel leaf qdisc for flow isolation and bufferbloat protection
		if err = self.addLeafQdisc(classid); err != nil {
			log.Printf("Warning: Failed to add fq_codel to class %s: %v", classid.String(), err)
		}

		self.classList = append(self.classList, klass)
		return nil, nil

	})

	return err
}

func (self *TcClassMgr) ChangeClass(parent *TcClass, classid TcClassId, down Kbit, up Kbit, ceilDown Kbit, ceilUp Kbit) error {
	_, err := tcClassQue.Exec("TcClassMgr.ChangeClass", func() (any, error) {
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
	_, err := tcClassQue.Exec("TcClassMgr.DeleteClass", func() (any, error) {
		for i, klass := range self.classList {
			if klass.ClassId == classid {
				// Remove fq_codel leaf qdisc before deleting class
				self.delLeafQdisc(classid)

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

	if err = cmd.Exec(fmt.Sprintf(`tc class %s dev %s parent %s classid %s htb rate %dkbit ceil %dkbit burst 15k cburst 15k`, action, self.dev, parentid, classid, klass.MinDown, klass.CeilDown), nil); err != nil {
		return err
	}

	if ifbutil.IsIfbSupported() {
		if err = cmd.Exec(fmt.Sprintf(`tc class %s dev %s parent %s classid %s htb rate %dkbit ceil %dkbit burst 15k cburst 15k`, action, ifb, parentid, classid, klass.MinUp, klass.CeilUp), nil); err != nil {
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

// rootTcClassLocked returns RootTcClass without acquiring lock (caller must hold lock)
func (self *TcClassMgr) rootTcClassLocked(download, upload Kbit) *TcClass {
	return &TcClass{
		ClassId:  TcClassIdRoot,
		MinDown:  download,
		MinUp:    upload,
		CeilDown: download,
		CeilUp:   upload,
	}
}

func (self *TcClassMgr) RootTcClass() *TcClass {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.rootTcClassLocked(self.download, self.upload)
}

func (self *TcClassMgr) DefaultTcClass() *TcClass {
	self.mu.RLock()
	download := self.download
	upload := self.upload
	root := self.rootTcClassLocked(download, upload)
	self.mu.RUnlock()

	minDown := download / 2
	minUp := upload / 2
	return NewTcClass(root, TcClassIdDefault, minDown, minUp, download, upload)
}

func (self *TcClassMgr) UserTcClass() *TcClass {
	self.mu.RLock()
	download := self.download
	upload := self.upload
	root := self.rootTcClassLocked(download, upload)
	self.mu.RUnlock()

	minDown := download / 2
	minUp := upload / 2
	return NewTcClass(root, TcClassIdUser, minDown, minUp, download, upload)
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

// Optimal fq_codel parameters based on bufferbloat.net research
// These values work well for links from 1 Mbps to 1 Gbps
const (
	fqCodelTarget   = "5ms"   // Target queue delay
	fqCodelInterval = "100ms" // CoDel control interval
	fqCodelLimit    = 1000    // Maximum packets in queue
	fqCodelQuantum  = 1514    // MTU-sized quantum for Ethernet
)

// addLeafQdisc adds fq_codel as leaf qdisc to eliminate bufferbloat.
// fq_codel provides:
// - Flow isolation (1024 queues) - prevents single flow from monopolizing bandwidth
// - CoDel AQM - keeps queue delays low (target 5ms)
// - Sparse flow priority - VoIP/gaming get priority over bulk transfers
// - ECN support - marks instead of dropping when possible
func (self *TcClassMgr) addLeafQdisc(classid TcClassId) error {
	dev := self.dev
	ifb := ifbName(dev)
	classidStr := classid.String()

	// Leaf qdisc handle derived from classid
	// Example: classid 1:100 → leaf handle 100:
	leafHandle := fmt.Sprintf("%d:", classid)

	// Build fq_codel command with hardcoded optimal parameters
	fqCmd := fmt.Sprintf(
		"tc qdisc add dev %%s parent %s handle %s fq_codel limit %d target %s interval %s quantum %d ecn",
		classidStr,
		leafHandle,
		fqCodelLimit,
		fqCodelTarget,
		fqCodelInterval,
		fqCodelQuantum,
	)

	// Add to egress interface (download shaping)
	if err := cmd.Exec(fmt.Sprintf(fqCmd, dev), nil); err != nil {
		return fmt.Errorf("failed to add fq_codel to %s class %s: %w", dev, classidStr, err)
	}

	// Add to IFB interface (upload shaping via ingress redirection)
	if ifbutil.IsIfbSupported() {
		if err := cmd.Exec(fmt.Sprintf(fqCmd, ifb), nil); err != nil {
			// Log but don't fail - IFB is optional enhancement
			log.Printf("Warning: failed to add fq_codel to %s class %s: %v", ifb, classidStr, err)
		}
	}

	log.Printf("Added fq_codel to class %s (target=%s, limit=%d, quantum=%d, ECN=true)",
		classidStr, fqCodelTarget, fqCodelLimit, fqCodelQuantum)

	return nil
}

// delLeafQdisc removes fq_codel leaf qdisc from a class during cleanup
func (self *TcClassMgr) delLeafQdisc(classid TcClassId) {
	dev := self.dev
	ifb := ifbName(dev)
	classidStr := classid.String()
	leafHandle := fmt.Sprintf("%d:", classid)

	// Delete from egress (ignore errors - qdisc may not exist)
	cmd.Exec(fmt.Sprintf("tc qdisc del dev %s parent %s handle %s", dev, classidStr, leafHandle), nil)

	// Delete from IFB ingress
	if ifbutil.IsIfbSupported() {
		cmd.Exec(fmt.Sprintf("tc qdisc del dev %s parent %s handle %s", ifb, classidStr, leafHandle), nil)
	}
}
