package sdkforms

type ListOption struct {
	Label string
	Value interface{}
}

type ListField struct {
	Name       string
	Label      string
	Type       string
	Multiple   bool
	Options    func() []ListOption
	DefaultVal interface{}
}

func (f ListField) GetName() string {
	return f.Name
}

func (f ListField) GetLabel() string {
	return f.Label
}

func (f ListField) GetType() string {
	return FormFieldTypeList
}

func (f ListField) GetDefaultVal() interface{} {
	return f.DefaultVal
}
