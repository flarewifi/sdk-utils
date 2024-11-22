package formsutl

import (
	"errors"
	"fmt"
	"net/url"
	sdkforms "sdk/api/forms"
	"strconv"
)

var (
	ErrNotBasicType = fmt.Errorf("field type is not a basic type, e.g. string, integer, decimal, bool")
)

func ParseBasicValue(fld sdkforms.IFormField, valstr []string) (val interface{}, err error) {
	switch fld.GetType() {
	case sdkforms.FormFieldTypeText:
		if len(valstr) < 1 {
			return "", nil
		}
		val = valstr[0]

	case sdkforms.FormFieldTypeInteger:
		if len(valstr) < 1 {
			return 0, nil
		}
		val, err = strconv.ParseInt(valstr[0], 10, 64)
		if err != nil {
			return 0, nil
		}
	case sdkforms.FormFieldTypeDecimal:
		if len(valstr) < 1 {
			return 0.0, nil
		}
		val, err = strconv.ParseFloat(valstr[0], 64)
		if err != nil {
			return 0, nil
		}
	case sdkforms.FormFieldTypeBoolean:
		if len(valstr) < 1 {
			return false, nil
		}
		val, err = strconv.ParseBool(valstr[0])
		if err != nil {
			return false, nil
		}
	default:
		err = ErrNotBasicType
	}
	return
}

func ParseListFieldValue(fld sdkforms.IFormField, valstr []string) (val interface{}, err error) {
	listField, ok := fld.(sdkforms.ListField)
	if !ok {
		err = fmt.Errorf("field %s is not a list field", fld.GetName())
		return
	}

	if valstr == nil {
		return GetTypeDefault(fld), nil
	}

	switch listField.Type {

	case sdkforms.FormFieldTypeText:
		vals := valstr
		val = valstr
		if !listField.Multiple {
			if len(vals) > 0 {
				val = vals[0]
				return
			}
			val = ""
		}
		return

	case sdkforms.FormFieldTypeInteger:
		vals := make([]int64, len(valstr))
		for i, v := range valstr {
			vals[i], err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, nil
			}
		}
		val = vals
		if !listField.Multiple {
			if len(vals) > 0 {
				val = vals[0]
				return
			}
			val = 0
		}
		return

	case sdkforms.FormFieldTypeDecimal:
		vals := make([]float64, len(valstr))
		for i, v := range valstr {
			vals[i], err = strconv.ParseFloat(v, 64)
			if err != nil {
				return 0, nil
			}
		}
		val = vals
		if !listField.Multiple {
			if len(vals) > 0 {
				val = vals[0]
				return
			}
			val = 0.0
		}
		return

	case sdkforms.FormFieldTypeBoolean:
		vals := make([]bool, len(valstr))
		for i, v := range valstr {
			vals[i], err = strconv.ParseBool(v)
			if err != nil {
				return false, nil
			}
		}
		val = vals
		if !listField.Multiple {
			if len(vals) > 0 {
				val = vals[0]
				return
			}
			val = false
		}
		return

	default:
		err = errors.New(fmt.Sprintf("%s default value %s is not supported list field", fld.GetName(), listField.Type))
	}

	return
}

func ParseMultiFieldValue(sec sdkforms.FormSection, f sdkforms.IFormField, form url.Values) (val [][]sdkforms.FieldData, err error) {
	fld, ok := f.(sdkforms.MultiField)
	if !ok {
		err = errors.New(fmt.Sprintf("field %s in section %s is not a multi-field", f.GetName(), sec.Name))
		return
	}

	columns := fld.Columns()
	if len(columns) < 1 {
		err = errors.New(fmt.Sprintf("multi-field %s in section %s has no columns", fld.Name, sec.Name))
		return
	}

	col1 := sec.Name + ":" + fld.Name + ":" + columns[0].Name
	numRows := len(form[col1])

	vals := make([][]sdkforms.FieldData, numRows)

	for ridx := 0; ridx < numRows; ridx++ {
		row := make([]sdkforms.FieldData, len(columns))
		for cidx, colfld := range columns {
			var value interface{}

			inputName := sec.Name + ":" + fld.Name + ":" + colfld.Name
			colarr := form[inputName]

			switch colfld.GetType() {

			case sdkforms.FormFieldTypeText,
				sdkforms.FormFieldTypeInteger,
				sdkforms.FormFieldTypeDecimal,
				sdkforms.FormFieldTypeBoolean:

				if ridx >= len(colarr) {
					value = GetTypeDefault(colfld)
					break
				}

				valstr := colarr[ridx]
				value, err = ParseBasicValue(colfld, []string{valstr})
				if err != nil {
					return nil, err
				}

			default:
				err = errors.New(fmt.Sprintf("unsupported list field type %s", colfld.GetType()))
				return
			}

			row[cidx] = sdkforms.FieldData{
				Name:  colfld.GetName(),
				Value: value,
			}
		}

		vals[ridx] = row
	}

	return vals, nil

}

func GetTypeDefault(fld sdkforms.IFormField) interface{} {
	switch fld.GetType() {

	case sdkforms.FormFieldTypeText,
		sdkforms.FormFieldTypeInteger,
		sdkforms.FormFieldTypeDecimal,
		sdkforms.FormFieldTypeBoolean:
		return GetBasicTypeDefault(fld.GetType())

	case sdkforms.FormFieldTypeList:
		lsfld := fld.(sdkforms.ListField)
		if lsfld.Multiple {
			return []interface{}{}
		} else {
			return GetBasicTypeDefault(fld.GetType())
		}

	case sdkforms.FormFieldTypeMulti:
		return map[string]interface{}{}

	default:
		return nil
	}
}

func GetBasicTypeDefault(t string) interface{} {
	switch t {
	case sdkforms.FormFieldTypeText:
		return ""
	case sdkforms.FormFieldTypeInteger:
		return 0
	case sdkforms.FormFieldTypeDecimal:
		return 0.0
	case sdkforms.FormFieldTypeBoolean:
		return false
	default:
		return nil
	}
}
