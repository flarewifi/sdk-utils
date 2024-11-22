package sdkforms

type IntegerField struct {
	Name       string
	Label      string
	DefaultVal int
}

func (f IntegerField) GetName() string {
	return f.Name
}

func (f IntegerField) GetLabel() string {
	return f.Label
}

func (f IntegerField) GetType() string {
	return FormFieldTypeInteger
}

func (f IntegerField) GetDefaultVal() interface{} {
	return f.DefaultVal
}
