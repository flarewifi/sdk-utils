package sdkapi

type FormFileField struct {
	Name      string
	Label     string
	ValueFn   func() []string
	Required  bool
	Multiple  bool
	MinFiles  int
	MaxFiles  int
	MinSizeMb int
	MaxSizeMb int
	Accept    []string
}

func (f FormFileField) GetName() string {
	return f.Name
}

func (f FormFileField) GetLabel() string {
	return f.Label
}

func (f FormFileField) GetType() string {
	return FormFieldTypeFile
}

func (f FormFileField) GetValue() interface{} {
	if f.ValueFn != nil {
		return f.ValueFn()
	}
	return []string{}
}

func (f FormFileField) IsRequired() bool {
	return f.Required
}
