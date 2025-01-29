package plugins

import (
	"errors"
	"fmt"
	"reflect"
	sdkapi "sdk/api"
)

type FormMultiFieldData struct {
	Fields [][]sdkapi.FormFieldData `json:"fields"`
}

func (f FormMultiFieldData) NumRows() int {
	return len(f.Fields)
}

func (f FormMultiFieldData) GetValue(row int, name string) (val interface{}, err error) {
	r := f.Fields[row]
	if r == nil {
		return "", errors.New(fmt.Sprintf("row %d not found", row))
	}

	for _, field := range r {
		if field.Name == name {
			return field.Value, nil
		}
	}

	return "", errors.New(fmt.Sprintf("field %s not found in multi-field", name))
}

func (f FormMultiFieldData) GetStringValue(row int, name string) (val string, err error) {
	v, err := f.GetValue(row, name)
	if err != nil {
		return "", err
	}

	val, ok := v.(string)
	if !ok {
		return "", errors.New(fmt.Sprintf("field %s in row %d in multi-field is not a string, instead %T", name, row, v))
	}

	return val, nil
}

func (f FormMultiFieldData) GetIntValue(row int, name string) (val int64, err error) {
	v, err := f.GetValue(row, name)
	if err != nil {
		return 0, err
	}

	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Float64:
		return int64(v.(float64)), nil
	case reflect.Int64:
		return v.(int64), nil
	default:
		return 0, nil
	}
}

func (f FormMultiFieldData) GetFloatValue(row int, name string) (val float64, err error) {
	v, err := f.GetValue(row, name)
	if err != nil {
		return 0, err
	}

	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Float64:
		return v.(float64), nil
	case reflect.Int:
		return float64(v.(int)), nil
	default:
		return 0, nil
	}
}

func (f FormMultiFieldData) GetBoolValue(row int, name string) (val bool, err error) {
	v, err := f.GetValue(row, name)
	if err != nil {
		return
	}

	val, ok := v.(bool)
	if !ok {
		err = errors.New(fmt.Sprintf("field %s in row %d in multi-field is not a boolean", name, row))
		return
	}

	return val, nil
}
