package api

import (
	formsview "core/resources/views/forms/bootstrap5"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	sdkapi "sdk/api"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	ErrFieldMulti   = errors.New("field type is multifield")
	ErrNotBasicType = fmt.Errorf("field type is not a basic type, e.g. string, integer, decimal, bool")
)

func NewHttpForm(api *PluginApi, form sdkapi.HttpForm) *HttpFormInstance {
	return &HttpFormInstance{
		api:       api,
		form:      form,
		validator: NewHTTPFormValidator(api),
	}
}

type HttpFormInstance struct {
	api       *PluginApi
	form      sdkapi.HttpForm
	data      []sdkapi.SectionData
	validator *HTTPFormValidator
}

func (self *HttpFormInstance) GetTemplate(r *http.Request) templ.Component {
	if strings.TrimSpace(self.form.SubmitLabel) == "" {
		self.form.SubmitLabel = "Submit"
	}

	errorMap, valueMap := self.validator.GetValidatedValues(r, self)
	return formsview.HtmlForm(&formsview.HtmlFormConfig{
		PluginAPI:  self.api,
		Form:       self,
		CSRFTag:    self.api.HttpAPI.Helpers().CsrfHtmlTag(r),
		SubmitURL:  self.api.HttpAPI.httpRouter.UrlForRoute(sdkapi.PluginRouteName(self.form.CallbackRoute)),
		SubmitText: self.form.SubmitLabel,
		ValueMap:   valueMap,
		ErrorMap:   errorMap,
	})
}

func (self *HttpFormInstance) GetSection(section string) (sdkapi.IFormSection, bool) {
	for _, s := range self.form.Sections {
		if s.Name == section {
			return NewFormSection(self, s), true
		}
	}
	return nil, false
}

func (self *HttpFormInstance) GetSections() []sdkapi.IFormSection {
	sections := make([]sdkapi.IFormSection, len(self.form.Sections))
	for i, s := range self.form.Sections {
		sections[i] = NewFormSection(self, s)
	}
	return sections
}

func (self *HttpFormInstance) ParseForm(w http.ResponseWriter, r *http.Request) (err error) {
	var validationError error
	if err := r.ParseForm(); err != nil {
		return err
	}

	parsedData := make([]sdkapi.SectionData, len(self.form.Sections))
	for sidx, sec := range self.form.Sections {
		sectionData := sdkapi.SectionData{
			Name:   sec.Name,
			Fields: make([]sdkapi.FormFieldData, len(sec.Fields)),
		}

		for fidx, fld := range sec.Fields {
			field := sdkapi.FormFieldData{Name: fld.GetName()}
			valstr := r.Form[sec.Name+":"+fld.GetName()]

			switch fld.GetType() {
			case sdkapi.FormFieldTypeString,
				sdkapi.FormFieldTypeText,
				sdkapi.FormFieldTypeInteger,
				sdkapi.FormFieldTypeDecimal,
				sdkapi.FormFieldTypeBoolean:
				field.Value, err = ParseBasicValue(fld, valstr)
				if err != nil {
					field.Value = fld.GetValue()
				}

			case sdkapi.FormFieldTypeList:
				field.Value, err = ParseListFieldValue(fld, valstr)
				if err != nil {
					field.Value = fld.GetValue()
				}

			case sdkapi.FormFieldTypeMulti:
				val, err := ParseMultiFieldValue(sec, fld, r.Form)
				if err != nil {
					mfld, ok := fld.(sdkapi.FormMultiField)
					if !ok {
						return fmt.Errorf("section %s, field %s type is not multifield, instead %T", sec, fld.GetName(), fld)
					}

					fldvals := mfld.GetValue()
					mfldval, ok := fldvals.([][]sdkapi.FormFieldData)
					if !ok {
						return fmt.Errorf("section %s, field %s value is not a slice of sdkapi.FieldData, instead %T", sec, fld.GetName(), fldvals)
					}
					val = mfldval
				}
				field.Value = val

			case sdkapi.FormFieldTypeFile:
				ffld, ok := fld.(sdkapi.FormFileField)
				if !ok {
					return fmt.Errorf("section %s, field %s type is not form field, instead %T", sec, fld.GetName(), fld)
				}

				val, err := ParseFile(r, sec, &ffld)
				if err != nil {
					log.Println("error parsing form file: ", err)
					field.Value = ffld.GetValue()
				}

				field.Value = val

				// We'll discard previously uploaded file cookies every new parsing call.
				startIndex := len(field.Value.([]string))
				self.validator.DeletePreviousFileInputCookies(w, r, sec.Name, ffld, startIndex)

			default:
				return errors.New("invalid field type" + fld.GetType())
			}

			if field.Value == nil {
				field.Value = GetTypeDefault(fld)
			}

			valErr := self.validator.ValidateFormField(w, sec, fld, field.Value)
			if valErr != nil {
				// We'll handle the error after all form fields are validated.
				validationError = valErr
			}

			sectionData.Fields[fidx] = field
		}

		parsedData[sidx] = sectionData
	}

	self.data = parsedData

	if validationError == nil {
		// Delete cookies created during validation if parsing is successful.
		self.validator.DeleteAllFormCookies(w, r, self)
	}

	return validationError
}

func (self *HttpFormInstance) GetStringValue(section string, field string) (val string, err error) {
	v, err := self.getFieldValue(section, field)
	if err != nil {
		return val, err
	}
	str, ok := v.(string)
	if !ok {
		return val, errors.New(fmt.Sprintf("section %s field %s is not a string, instead %T", section, field, v))
	}
	return str, nil
}

func (self *HttpFormInstance) GetStringValues(section string, field string) (val []string, err error) {
	ivals, err := self.getFieldValues(section, field)
	if err != nil {
		return nil, err
	}

	val, ok := ivals.([]string)
	if !ok {
		return nil, errors.New(fmt.Sprintf("section %s, field %s is not a slice of strings", section, field))
	}

	return val, nil
}

func (self *HttpFormInstance) GetIntValue(section string, field string) (val int64, err error) {
	v, err := self.getFieldValue(section, field)
	if err != nil {
		return
	}
	if v == nil {
		return
	}

	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Float32, reflect.Float64:
		return int64(v.(float64)), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, ok := v.(int); ok {
			return int64(val), nil
		}

		return v.(int64), nil
	}
	return val, errors.New(fmt.Sprintf("section %s, field %s is not an int", section, field))
}

func (self *HttpFormInstance) GetIntValues(section string, field string) (val []int64, err error) {
	ivals, err := self.getFieldValues(section, field)
	if err != nil {
		return
	}

	t := reflect.TypeOf(ivals).Elem()
	val = []int64{}

	switch t.Kind() {
	case reflect.Int64:
		vals := ivals.([]int64)
		return vals, nil
	default:
		return nil, errors.New(fmt.Sprintf("section %s, field %s is not a slice of int64", section, field))
	}
}

func (self *HttpFormInstance) GetFloatValue(section string, field string) (val float64, err error) {
	v, err := self.getFieldValue(section, field)
	if err != nil {
		return
	}
	if val, ok := v.(float64); ok {
		return val, nil
	}
	return val, errors.New(fmt.Sprintf("section %s, field %s is not a float64", section, field))
}

func (self *HttpFormInstance) GetFloatValues(section string, field string) (val []float64, err error) {
	ivals, err := self.getFieldValues(section, field)
	if err != nil {
		return
	}

	t := reflect.TypeOf(ivals).Elem()
	switch t.Kind() {
	case reflect.Float64:
		vals := ivals.([]float64)
		return vals, nil
	default:
		return nil, errors.New(fmt.Sprintf("section %s, field %s is not a slice of float64", section, field))
	}
}

func (self *HttpFormInstance) GetBoolValue(section string, field string) (val bool, err error) {
	v, err := self.getFieldValue(section, field)
	if err != nil {
		return
	}
	if val, ok := v.(bool); ok {
		return val, nil
	}
	return false, errors.New(fmt.Sprintf("section %s, field %s is not a boolean", section, field))
}

func (self *HttpFormInstance) GetBoolValues(section string, field string) (val []bool, err error) {
	ivals, err := self.getFieldValues(section, field)
	if err != nil {
		return
	}

	t := reflect.TypeOf(ivals).Elem()
	switch t.Kind() {
	case reflect.Bool:
		vals := ivals.([]bool)
		return vals, nil
	default:
		return nil, errors.New(fmt.Sprintf("section %s, field %s is not a slice of boolean", section, field))
	}
}

func (self *HttpFormInstance) GetMultiField(section string, field string) (val sdkapi.IFormMultiField, err error) {
	v, err := self.getFieldValue(section, field)
	if err != nil {
		return
	}

	data, ok := v.([][]sdkapi.FormFieldData)
	if !ok {
		return val, errors.New(fmt.Sprintf("section %s, field %s value is not [][]sdkapi.FormFieldData, instead %T", section, field, v))
	}

	return FormMultiFieldData{
		Fields: data,
	}, nil
}

func (self *HttpFormInstance) GetFilePath(section string, field string) (string, error) {
	urls, err := self.GetFilePaths(section, field)
	if len(urls) > 0 {
		return urls[0], nil
	}

	return "", err
}

func (self *HttpFormInstance) GetFilePaths(section string, field string) ([]string, error) {
	filePaths := []string{}
	uploadDir := filepath.Join(sdkutils.PathTmpDir, "uploads", section, field)

	if err := sdkutils.FsListFiles(uploadDir, &filePaths, false); err != nil {
		log.Println("error listing files from: ", uploadDir)

		ivals, err := self.getFieldValues(section, field)
		if err != nil {
			return []string{}, fmt.Errorf("unabel to get field values: %w", err)
		}

		if ivals == nil {
			return []string{}, nil
		}

		return ivals.([]string), nil
	}

	return filePaths, nil
}

func (self *HttpFormInstance) getSection(section string) (sec sdkapi.FormSection, ok bool) {
	for _, s := range self.form.Sections {
		if s.Name == section {
			return s, true
		}
	}
	return
}

func (self *HttpFormInstance) getField(section string, field string) (f sdkapi.IFormField, ok bool) {
	for _, s := range self.form.Sections {
		if s.Name == section {
			for _, fld := range s.Fields {
				if fld.GetName() == field {
					return fld, true
				}
			}
		}
	}
	return
}

func (self *HttpFormInstance) getParsedSection(section string) (sec sdkapi.SectionData, ok bool) {
	data := self.data
	if data == nil {
		return
	}

	for _, s := range data {
		if s.Name == section {
			return s, true
		}
	}
	return
}

func (self *HttpFormInstance) getParsedField(section string, field string) (fld sdkapi.FormFieldData, ok bool) {
	if s, ok := self.getParsedSection(section); ok {
		for _, f := range s.Fields {
			if f.Name == field {
				return f, true
			}
		}
	}
	return
}

func (self *HttpFormInstance) getParsedFieldValue(section string, field string) (val interface{}, ok bool) {
	if f, ok := self.getParsedField(section, field); ok {
		return f.Value, true
	}
	return
}

func (self *HttpFormInstance) getFieldValue(section string, field string) (val interface{}, err error) {
	if self.data == nil {
		fld, ok := self.getField(section, field)
		if !ok {
			return nil, errors.New(fmt.Sprintf("section %s, field %s value not found", section, field))
		}
		return fld.GetValue(), nil
	}

	if v, ok := self.getParsedFieldValue(section, field); ok {
		return v, nil
	}

	return nil, errors.New(fmt.Sprintf("section %s, field %s value not found", section, field))
}

func (self *HttpFormInstance) getFieldValues(section string, field string) (val interface{}, err error) {
	var ok bool
	if self.data == nil {
		fld, ok := self.getField(section, field)
		if !ok {
			return nil, errors.New(fmt.Sprintf("section %s, field %s value not found", section, field))
		}
		val = fld.GetValue()
	} else {
		val, ok = self.getParsedFieldValue(section, field)
		if !ok {
			return nil, errors.New(fmt.Sprintf("section %s, field %s values not found", section, field))
		}
	}

	if reflect.TypeOf(val).Kind() != reflect.Slice {
		return nil, errors.New(fmt.Sprintf("section %s, field %s values is not a slice", section, field))
	}

	return val, nil
}

// ----- Parser functions ----
func ParseBasicValue(fld sdkapi.IFormField, valstr []string) (val interface{}, err error) {
	switch fld.GetType() {
	case sdkapi.FormFieldTypeString,
		sdkapi.FormFieldTypeText:

		stringFld, ok := fld.(sdkapi.FormStringField)
		if ok && stringFld.IsReadOnly {
			return stringFld.GetValue(), nil
		}

		if len(valstr) < 1 {
			return "", nil
		}
		val = valstr[0]
	case sdkapi.FormFieldTypeInteger:
		if len(valstr) < 1 {
			return 0, nil
		}
		val, err = strconv.ParseInt(valstr[0], 10, 64)
		if err != nil {
			return 0, nil
		}
	case sdkapi.FormFieldTypeDecimal:
		if len(valstr) < 1 {
			return float64(0.0), nil
		}
		val, err = strconv.ParseFloat(valstr[0], 64)
		if err != nil {
			return float64(0.0), nil
		}
	case sdkapi.FormFieldTypeBoolean:
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

func ParseListFieldValue(fld sdkapi.IFormField, valstr []string) (val interface{}, err error) {
	listField, ok := fld.(sdkapi.FormListField)
	if !ok {
		err = fmt.Errorf("field %s is not a list field", fld.GetName())
		return
	}

	if valstr == nil {
		return GetTypeDefault(fld), nil
	}

	switch listField.Type {
	case sdkapi.FormFieldTypeString,
		sdkapi.FormFieldTypeText:
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

	case sdkapi.FormFieldTypeInteger:
		vals := make([]int64, len(valstr))
		for i, v := range valstr {
			vals[i], err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing error: %w", err)
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

	case sdkapi.FormFieldTypeDecimal:
		vals := make([]float64, len(valstr))
		for i, v := range valstr {
			vals[i], err = strconv.ParseFloat(v, 64)
			if err != nil {
				return 0, fmt.Errorf("parse float error: %w", err)
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

	case sdkapi.FormFieldTypeBoolean:
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

func ParseMultiFieldValue(sec sdkapi.FormSection, f sdkapi.IFormField, form url.Values) (val [][]sdkapi.FormFieldData, err error) {
	fld, ok := f.(sdkapi.FormMultiField)
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

	vals := make([][]sdkapi.FormFieldData, numRows)

	for ridx := 0; ridx < numRows; ridx++ {
		row := make([]sdkapi.FormFieldData, len(columns))
		for cidx, colfld := range columns {
			var value interface{}

			inputName := sec.Name + ":" + fld.Name + ":" + colfld.Name
			colarr := form[inputName]

			switch colfld.GetType() {

			case sdkapi.FormFieldTypeString,
				sdkapi.FormFieldTypeText,
				sdkapi.FormFieldTypeInteger,
				sdkapi.FormFieldTypeDecimal,
				sdkapi.FormFieldTypeBoolean:

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

			row[cidx] = sdkapi.FormFieldData{
				Name:  colfld.GetName(),
				Value: value,
			}
		}

		vals[ridx] = row
	}

	return vals, nil
}

func ParseFile(r *http.Request, sec sdkapi.FormSection, fld *sdkapi.FormFileField) (urls []string, err error) {
	// Parse up to 32MB in memory, rest in temp files
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	files := r.MultipartForm.File[fmt.Sprintf("%s:%s", sec.Name, fld.GetName())]
	if len(files) == 0 {
		return getPreviouslyUploadedFiles(sec, fld)
	}

	uploadDir := filepath.Join(sdkutils.PathTmpDir, "uploads", sec.Name, fld.Name)
	if err := sdkutils.FsEmptyDir(uploadDir); err != nil {
		return nil, fmt.Errorf("ensure dir error: %w", err)
	}
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("%v: Failed to open file %s\n", err, fileHeader.Filename)
			continue
		}
		defer file.Close()

		filePath := filepath.Join(uploadDir, fileHeader.Filename)
		dst, err := os.Create(filePath)
		if err != nil {
			log.Printf("%v: Failed to save file %s\n", err, fileHeader.Filename)
			continue
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			log.Printf("%v: Failed to open file %s\n", err, fileHeader.Filename)
			continue
		}

		urls = append(urls, filePath)
	}

	return urls, nil
}

func getPreviouslyUploadedFiles(
	sec sdkapi.FormSection,
	fld *sdkapi.FormFileField,
) (urls []string, err error) {
	uploadDir := filepath.Join(sdkutils.PathTmpDir, "uploads", sec.Name, fld.Name)

	if err := sdkutils.FsListFiles(uploadDir, &urls, false); err != nil {
		return urls, fmt.Errorf("unable to get uploaded files: %w", err)
	}

	return urls, nil
}

func GetTypeDefault(fld sdkapi.IFormField) interface{} {
	switch fld.GetType() {

	case sdkapi.FormFieldTypeString,
		sdkapi.FormFieldTypeText,
		sdkapi.FormFieldTypeInteger,
		sdkapi.FormFieldTypeDecimal,
		sdkapi.FormFieldTypeBoolean:
		return GetBasicTypeDefault(fld.GetType())

	case sdkapi.FormFieldTypeList:
		lsfld := fld.(sdkapi.FormListField)
		if lsfld.Multiple {
			return []interface{}{}
		} else {
			return GetBasicTypeDefault(fld.GetType())
		}

	case sdkapi.FormFieldTypeMulti:
		return map[string]interface{}{}

	case sdkapi.FormFieldTypeFile:
		return []string{}

	default:
		return nil
	}
}

func GetBasicTypeDefault(t string) interface{} {
	switch t {
	case sdkapi.FormFieldTypeString,
		sdkapi.FormFieldTypeText:
		return ""
	case sdkapi.FormFieldTypeInteger:
		return int64(0)
	case sdkapi.FormFieldTypeDecimal:
		return float64(0.0)
	case sdkapi.FormFieldTypeBoolean:
		return false
	default:
		return nil
	}
}
