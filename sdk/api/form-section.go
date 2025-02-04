package sdkapi

type IFormSection interface {
	GetName() string
	GetLabel() string
	GetFields() []IFormField

	GetStringValue(name string) (string, error)
	GetStringValues(name string) ([]string, error)

	GetIntValue(name string) (int64, error)
	GetIntValues(name string) ([]int64, error)

	GetFloatValue(name string) (float64, error)
	GetFloatValues(name string) ([]float64, error)

	GetBoolValue(name string) (bool, error)
	GetBoolValues(name string) ([]bool, error)

	GetMultiField(name string) (IFormMultiField, error)
}
