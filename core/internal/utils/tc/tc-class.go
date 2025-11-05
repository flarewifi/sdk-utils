package tc

// Kbit type
type Kbit uint

func (kbit Kbit) ToMbit() Mbit {
	return Mbit(kbit / 1000)
}

func (kbit Kbit) ToUint() uint {
	return uint(kbit)
}

// Mbit type
type Mbit float64

func (mbit Mbit) ToKbit() Kbit {
	return Kbit(mbit * 1000)
}

func (mbit Mbit) ToUint() float64 {
	return float64(mbit)
}

// TcClass type
type TcClass struct {
	Parent   *TcClass
	ClassId  TcClassId
	MinDown  Kbit
	MinUp    Kbit
	CeilDown Kbit
	CeilUp   Kbit
}

func NewTcClass(parent *TcClass, classid TcClassId, d Kbit, u Kbit, cd Kbit, cu Kbit) *TcClass {
	return &TcClass{parent, classid, d, u, cd, cu}
}

func (c *TcClass) Sanitize() {
	if c.CeilDown == 0 {
		c.CeilDown = c.MinDown
	}
	if c.CeilUp == 0 {
		c.CeilUp = c.MinUp
	}
	if c.MinDown > c.CeilDown {
		c.MinDown = c.CeilDown
	}
	if c.MinUp > c.CeilUp {
		c.MinUp = c.CeilUp
	}

	// make sure MinDown, MinUp, CeilDown and CeilUp does not exceed parent's
	if c.Parent != nil {
		if c.MinDown > c.Parent.MinDown {
			c.MinDown = c.Parent.MinDown
		}
		if c.MinUp > c.Parent.MinUp {
			c.MinUp = c.Parent.MinUp
		}
		if c.CeilDown > c.Parent.CeilDown {
			c.CeilDown = c.Parent.CeilDown
		}
		if c.CeilUp > c.Parent.CeilUp {
			c.CeilUp = c.Parent.CeilUp
		}
	}
}
