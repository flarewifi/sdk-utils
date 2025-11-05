package api

import sdkapi "sdk/api"

func NewFormSection(form *HttpFormInstance, section sdkapi.FormSection) *FormSection {
	return &FormSection{
		form:    form,
		section: section,
	}
}

type FormSection struct {
	form    *HttpFormInstance
	section sdkapi.FormSection
}

func (f *FormSection) GetFormSection() sdkapi.FormSection {
	return f.section
}

func (f *FormSection) GetName() string {
	return f.section.Name
}

func (f *FormSection) GetLabel() string {
	return f.section.Label
}

func (f *FormSection) GetFields() []sdkapi.IFormField {
	return f.section.Fields
}

func (f *FormSection) GetBoolValue(name string) (bool, error) {
	return f.form.GetBoolValue(f.section.Name, name)
}

func (f *FormSection) GetBoolValues(name string) ([]bool, error) {
	return f.form.GetBoolValues(f.section.Name, name)
}

func (f *FormSection) GetFloatValue(name string) (float64, error) {
	return f.form.GetFloatValue(f.section.Name, name)
}

func (f *FormSection) GetFloatValues(name string) ([]float64, error) {
	return f.form.GetFloatValues(f.section.Name, name)
}

func (f *FormSection) GetIntValue(name string) (int64, error) {
	return f.form.GetIntValue(f.section.Name, name)
}

func (f *FormSection) GetIntValues(name string) ([]int64, error) {
	return f.form.GetIntValues(f.section.Name, name)
}

func (f *FormSection) GetMultiField(name string) (sdkapi.IFormMultiField, error) {
	return f.form.GetMultiField(f.section.Name, name)
}

func (f *FormSection) GetStringValue(name string) (string, error) {
	return f.form.GetStringValue(f.section.Name, name)
}

func (f *FormSection) GetStringValues(name string) ([]string, error) {
	return f.form.GetStringValues(f.section.Name, name)
}
