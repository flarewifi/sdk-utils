package sdkforms

const (
	FormFieldTypeText    string = "text"
	FormFieldTypeDecimal string = "decimal"
	FormFieldTypeInteger string = "int"
	FormFieldTypeBoolean string = "bool"
	FormFieldTypeList    string = "list"
	FormFieldTypeMulti   string = "multi"
)

type SectionData struct {
	Name   string      `json:"name"`
	Fields []FieldData `json:"fields"`
}

type FieldData struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type IFormField interface {
	GetName() string
	GetLabel() string
	GetType() string
	GetDefaultVal() interface{}
}

type FormSection struct {
	Name   string
	Fields []IFormField
}

type Form struct {
	Name          string
	CallbackRoute string
	Sections      []FormSection
}

type JsonData []JsonSection
