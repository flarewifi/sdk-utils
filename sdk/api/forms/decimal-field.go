package sdkforms

type DecimalField struct {
	Name       string
	Label      string
	Step       float64
	Precision  int
	DefaultVal float64
}

func (f DecimalField) GetName() string {
	return f.Name
}

func (f DecimalField) GetLabel() string {
	return f.Label
}

func (f DecimalField) GetType() string {
	return FormFieldTypeDecimal
}

func (f DecimalField) GetDefaultVal() interface{} {
	return f.DefaultVal
}
