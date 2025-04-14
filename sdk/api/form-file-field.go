package sdkapi

type FormFileField struct {
	Name      string          // name of the input field
	Label     string          // label for the input field
	ValueFn   func() []string // returns the URL paths of the files
	Required  bool            // file is required
	Multiple  bool            // accept multiple files, otherwise only 1
	MinFiles  int             // (applicable only to multiple files) minimum number of files
	MaxFiles  int             // (applicable only to multiple files) max number of files
	MinSizeMb int             // minimum bytes
	MaxSizeMb int             // maximum bytes
	Accept    []string        // file types to accept
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
