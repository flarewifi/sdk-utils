package sdkforms

type BooleanField struct {
	Name       string
	Label      string
	DefaultVal bool
}

func (f BooleanField) GetName() string {
	return f.Name
}

func (f BooleanField) GetLabel() string {
	return f.Label
}

func (f BooleanField) GetType() string {
	return FormFieldTypeBoolean
}

func (f BooleanField) GetDefaultVal() interface{} {
	return f.DefaultVal
}
