package sdkforms

type IMultiField interface {
	NumRows() int
	GetStringValue(row int, name string) (string, error)
	GetIntValue(row int, name string) (int, error)
	GetFloatValue(row int, name string) (float64, error)
	GetBoolValue(row int, name string) (bool, error)
}

type MultiFieldCol struct {
	Name       string
	Label      string
	Type       string
	DefaultVal interface{}
}

func (col MultiFieldCol) GetName() string {
	return col.Name
}

func (col MultiFieldCol) GetLabel() string {
	return col.Label
}

func (col MultiFieldCol) GetType() string {
	return col.Type
}

func (col MultiFieldCol) GetDefaultVal() interface{} {
	return col.DefaultVal
}

type MultiField struct {
	Name       string
	Label      string
	Columns    func() []MultiFieldCol
	DefaultVal MultiFieldData
}

func (f MultiField) GetName() string {
	return f.Name
}

func (f MultiField) GetLabel() string {
	return f.Label
}

func (f MultiField) GetType() string {
	return FormFieldTypeMulti
}

func (f MultiField) GetDefaultVal() interface{} {
	return f.DefaultVal
}
