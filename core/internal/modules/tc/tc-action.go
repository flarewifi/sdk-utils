package tc

type tcAction uint8

func (t tcAction) String() string {
	switch t {
	case tcActionAdd:
		return "add"
	case tcActionChange:
		return "change"
	case tcActionDelete:
		return "del"
	}
	return ""
}

const (
	tcActionAdd tcAction = iota
	tcActionChange
	tcActionDelete
)
