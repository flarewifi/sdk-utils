package formsutl

import (
	"fmt"
	sdkforms "sdk/api/forms"
	sdkhttp "sdk/api/http"

	sdkslices "github.com/flarehotspot/go-utils/slices"
)

func ToJson(form sdkhttp.IHttpForm, sections []sdkforms.FormSection) (sdkforms.JsonData, error) {
	data := make([]sdkforms.JsonSection, len(sections))

	for i, sec := range sections {
		secjson := sdkforms.JsonSection{
			Name:   sec.Name,
			Fields: make([]sdkforms.JsonField, len(sec.Fields)),
		}

		for j, fld := range sec.Fields {
			fldjson := sdkforms.JsonField{
				Name:  fld.GetName(),
				Label: fld.GetLabel(),
				Type:  fld.GetType(),
			}

			switch fld.GetType() {
			case sdkforms.FormFieldTypeText:
				val, err := form.GetStringValue(sec.Name, fld.GetName())
				if err != nil {
					return nil, err
				}
				fldjson.Value = val

			case sdkforms.FormFieldTypeInteger:
				val, err := form.GetIntValue(sec.Name, fld.GetName())
				if err != nil {
					return nil, err
				}
				fldjson.Value = val

			case sdkforms.FormFieldTypeDecimal:
				val, err := form.GetFloatValue(sec.Name, fld.GetName())
				if err != nil {
					return nil, err
				}
				fldjson.Value = val

			case sdkforms.FormFieldTypeBoolean:
				val, err := form.GetBoolValue(sec.Name, fld.GetName())
				if err != nil {
					return nil, err
				}
				fldjson.Value = val

			case sdkforms.FormFieldTypeList:
				listFld, ok := fld.(sdkforms.ListField)
				if !ok {
					return nil, fmt.Errorf("field %s is not a list-field", fld.GetName())
				}

				if listFld.Multiple {
					fldjson.ListMultiple = true

					switch listFld.Type {
					case sdkforms.FormFieldTypeText:
						val, err := form.GetStringValues(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val

					case sdkforms.FormFieldTypeInteger:
						val, err := form.GetIntValues(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val
					case sdkforms.FormFieldTypeDecimal:
						val, err := form.GetFloatValues(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val
					case sdkforms.FormFieldTypeBoolean:
						val, err := form.GetBoolValues(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val
					}
				} else {
					switch listFld.Type {
					case sdkforms.FormFieldTypeText:
						val, err := form.GetStringValue(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val
					case sdkforms.FormFieldTypeInteger:
						val, err := form.GetIntValue(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val
					case sdkforms.FormFieldTypeDecimal:
						val, err := form.GetFloatValue(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val
					case sdkforms.FormFieldTypeBoolean:
						val, err := form.GetBoolValue(sec.Name, fld.GetName())
						if err != nil {
							return nil, err
						}
						fldjson.Value = val
					}

				}

				opts := listFld.Options()
				fldjson.ListOptions = make([]sdkforms.JsonListOpt, len(opts))

				for i, opt := range opts {
					jsonListOpt := sdkforms.JsonListOpt{Label: opt.Label}

					switch listFld.Type {
					case sdkforms.FormFieldTypeText:
						jsonListOpt.Value = opt.Value.(string)
						if listFld.Multiple {
							vals := fldjson.Value.([]string)
							jsonListOpt.Selected = sdkslices.Contains(vals, jsonListOpt.Value)
						} else {
							jsonListOpt.Selected = fmt.Sprintf("%s", fldjson.Value) == fmt.Sprintf("%s", opt.Value)
						}

					case sdkforms.FormFieldTypeInteger:
						jsonListOpt.Value = fmt.Sprintf("%d", opt.Value.(int))
						if listFld.Multiple {
							vals := fldjson.Value.([]int)
							valStrs := make([]string, len(vals))
							for i, v := range vals {
								valStrs[i] = fmt.Sprintf("%d", v)
							}
							jsonListOpt.Selected = sdkslices.Contains(valStrs, jsonListOpt.Value)
						} else {
							jsonListOpt.Selected = fmt.Sprintf("%d", fldjson.Value) == fmt.Sprintf("%d", opt.Value)
						}

					case sdkforms.FormFieldTypeDecimal:
						jsonListOpt.Value = fmt.Sprintf("%.2f", opt.Value.(float64))
						if listFld.Multiple {
							vals := fldjson.Value.([]float64)
							valStrs := make([]string, len(vals))
							for i, v := range vals {
								valStrs[i] = fmt.Sprintf("%.2f", v)
							}
							jsonListOpt.Selected = sdkslices.Contains(valStrs, jsonListOpt.Value)
						} else {
							jsonListOpt.Selected = fmt.Sprintf("%.2f", fldjson.Value.(float64)) == fmt.Sprintf("%.2f", opt.Value)
						}

					case sdkforms.FormFieldTypeBoolean:
						jsonListOpt.Value = fmt.Sprintf("%t", opt.Value.(bool))
						if listFld.Multiple {
							vals := fldjson.Value.([]bool)
							valStrs := make([]string, len(vals))
							for i, v := range vals {
								valStrs[i] = fmt.Sprintf("%t", v)
							}
							jsonListOpt.Selected = sdkslices.Contains(valStrs, jsonListOpt.Value)
						} else {
							jsonListOpt.Selected = opt.Value.(bool) && fldjson.Value.(bool)
						}
					}

					fldjson.ListOptions[i] = jsonListOpt
				}

			case sdkforms.FormFieldTypeMulti:
				mfld, ok := fld.(sdkforms.MultiField)
				if !ok {
					return nil, fmt.Errorf("field %s is not a multi-field", fld.GetName())
				}

				mfval, err := form.GetMultiField(sec.Name, fld.GetName())
				if err != nil {
					return nil, err
				}

				fldjson.MultiColumns = make([]sdkforms.JsonMultiCol, len(mfld.Columns()))
				fldval := make([][]interface{}, mfval.NumRows())

				for i, colfld := range mfld.Columns() {
					fldjson.MultiColumns[i] = sdkforms.JsonMultiCol{
						Name:  colfld.Name,
						Label: colfld.Label,
						Type:  colfld.Type,
					}
				}

				for i := 0; i < mfval.NumRows(); i++ {
					rowdata := make([]interface{}, len(mfld.Columns()))
					for j, colfld := range mfld.Columns() {
						switch colfld.GetType() {
						case sdkforms.FormFieldTypeText:
							val, err := mfval.GetStringValue(i, colfld.GetName())
							if err != nil {
								return nil, err
							}
							rowdata[j] = val
						case sdkforms.FormFieldTypeInteger:
							val, err := mfval.GetIntValue(i, colfld.GetName())
							if err != nil {
								return nil, err
							}
							rowdata[j] = val
						case sdkforms.FormFieldTypeDecimal:
							val, err := mfval.GetFloatValue(i, colfld.GetName())
							if err != nil {
								return nil, err
							}
							rowdata[j] = val
						case sdkforms.FormFieldTypeBoolean:
							val, err := mfval.GetBoolValue(i, colfld.GetName())
							if err != nil {
								return nil, err
							}
							rowdata[j] = val
						}
					}
					fldval[i] = rowdata
				}

				fldjson.Value = fldval

			default:
				return nil, fmt.Errorf("field %s is unknown type %s", fld.GetName(), fld.GetType())
			}

			secjson.Fields[j] = fldjson
		}

		data[i] = secjson
	}

	return data, nil
}
