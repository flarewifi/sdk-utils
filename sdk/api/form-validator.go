package sdkapi

type FormWithValidator struct {
	FormName       string
	FormValidators []FormValidator
}

type FormValidator struct {
	FieldName  string
	FieldLabel string
	FieldType  string
	FieldRules FormFieldRules
}

type FormFieldRules struct {
	Required bool
	Minimum  int
	Maximum  int
}
