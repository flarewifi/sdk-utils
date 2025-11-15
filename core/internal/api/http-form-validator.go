package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	sdkapi "sdk/api"
	"strconv"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewHTTPFormValidator(api *PluginApi) *HTTPFormValidator {
	return &HTTPFormValidator{
		api: api,
	}
}

var errorCookieName = "%v_field_error"
var valueCookieName = "%v_field_value"

type HTTPFormValidator struct {
	api *PluginApi
}

// FormValues implements sdkapi.IFormValues
type FormValues struct {
	values map[string]string
}

// GetStringValue implements sdkapi.IFormValues.GetStringValue
func (fv *FormValues) GetStringValue(name string) (string, error) {
	val, ok := fv.values[name]
	if !ok {
		return "", fmt.Errorf("field %s not found", name)
	}
	return val, nil
}

// GetIntValue implements sdkapi.IFormValues.GetIntValue
func (fv *FormValues) GetIntValue(name string) (int64, error) {
	val, ok := fv.values[name]
	if !ok {
		return 0, fmt.Errorf("field %s not found", name)
	}
	intVal, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("field %s is not a valid integer: %w", name, err)
	}
	return intVal, nil
}

// GetFloatValue implements sdkapi.IFormValues.GetFloatValue
func (fv *FormValues) GetFloatValue(name string) (float64, error) {
	val, ok := fv.values[name]
	if !ok {
		return 0, fmt.Errorf("field %s not found", name)
	}
	floatVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("field %s is not a valid float: %w", name, err)
	}
	return floatVal, nil
}

// GetBoolValue implements sdkapi.IFormValues.GetBoolValue
func (fv *FormValues) GetBoolValue(name string) (bool, error) {
	val, ok := fv.values[name]
	if !ok {
		return false, fmt.Errorf("field %s not found", name)
	}
	// Handle common boolean representations
	lowerVal := strings.ToLower(strings.TrimSpace(val))
	switch lowerVal {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off", "":
		return false, nil
	default:
		return false, fmt.Errorf("field %s is not a valid boolean: %s", name, val)
	}
}

// GetFilePath implements sdkapi.IFormValues.GetFilePath
func (fv *FormValues) GetFilePath(name string) (string, error) {
	val, ok := fv.values[name]
	if !ok {
		return "", fmt.Errorf("field %s not found", name)
	}
	return val, nil
}

func (validtr *HTTPFormValidator) validateInteger(val any, min, max int, label string) (errStr string) {
	valInt, ok := val.(int)
	if !ok {
		valInt = int(val.(int64))
	}

	if valInt < min {
		errStr = validtr.api.Translate("error", "Input value does not meet the required minimum", "label", label, "min", min)
	}

	if max != 0 && valInt > max {
		errStr = validtr.api.Translate("error", "Input value exceeds the maximum allowed", "label", label, "max", max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateDecimal(val any, min, max int, label string) (errStr string) {
	fval := val.(float64)
	if fval < float64(min) {
		errStr = validtr.api.Translate("error", "Input value does not meet the required minimum", "label", label, "min", min)
	}

	if float64(max) != 0 && fval > float64(max) {
		errStr = validtr.api.Translate("error", "Input value exceeds the maximum allowed", "label", label, "max", max)
	}

	return errStr
}

func (validtr *HTTPFormValidator) validateString(required bool, val, label string, min, max int) (errStr string) {
	valStr := strings.TrimSpace(fmt.Sprint(val))

	if required && valStr == "" {
		errStr = validtr.api.Translate("error", "Input field is required", "label", label)
		return
	}

	if min > 0 && len(valStr) < min {
		errStr = validtr.api.Translate("error", "Input value does not meet the required minimum characters", "label", label, "min", min)
	}

	if max != 0 && len(valStr) > max {
		errStr = validtr.api.Translate("error", "Input value exceeds the maximum allowed characters", "label", label, "max", max)
	}

	return errStr
}

// ValidateAndExtractForm validates form data and extracts values
func (validtr *HTTPFormValidator) ValidateAndExtractForm(w http.ResponseWriter, r *http.Request, form sdkapi.FormValidator) (sdkapi.IFormValues, error) {
	var validateErr error
	cookie := validtr.api.HttpAPI.Cookie()
	formValues := map[string]string{}

	// Check if we need to parse multipart form for file uploads
	hasFileFields := false
	for _, validator := range form.Validators {
		if validator.FieldType == sdkapi.FormFieldTypeFile {
			hasFileFields = true
			break
		}
	}

	if hasFileFields {
		// Parse multipart form with max 10MB
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}
	}

	for _, validator := range form.Validators {
		var (
			fieldName  = validator.FieldName
			fieldLabel = validator.FieldLabel
			fieldType  = validator.FieldType
			rules      = validator.FieldRules

			cookieName = fmt.Sprintf("%v-%v", form.Name, fieldName)

			val = r.FormValue(fieldName)
		)

		// Extract validation rules
		isRequired := false
		minimum := 0
		maximum := 0
		allowedExtensions := ""
		for _, rule := range rules {
			switch rule {
			case sdkapi.FormFieldRuleRequired:
				isRequired = true
			case sdkapi.FormFieldRuleFileExt:
				// This would need to be extracted from a more complex rule structure
				// For now, we'll skip this
			}
		}

		var errStr string
		switch fieldType {
		case sdkapi.FormFieldTypeFile:
			// Handle file upload
			filePath, err := validtr.handleFileUpload(r, fieldName, fieldLabel, isRequired, allowedExtensions)
			if err != nil {
				errStr = err.Error()
			} else if filePath != "" {
				formValues[fieldName] = filePath
			}

		case sdkapi.FormFieldTypeString:
			formValues[fieldName] = val
			errStr = validtr.validateString(isRequired, val, fieldLabel, minimum, maximum)

		case sdkapi.FormFieldTypeInteger:
			formValues[fieldName] = val
			if !isRequired && val == "" {
				// Clear any previous error
				cookie.DeleteCookie(w, cookieName)
				continue
			}
			_, err := strconv.Atoi(val)
			if err != nil && (isRequired || val != "") {
				errStr = validtr.api.Translate("error", "Input value must be a valid integer", "label", fieldLabel)
			} else if err == nil {
				valInt, _ := strconv.Atoi(val)
				errStr = validtr.validateInteger(valInt, minimum, maximum, fieldLabel)
			}

		case sdkapi.FormFieldTypeDecimal:
			formValues[fieldName] = val
			if !isRequired && val == "" {
				// Clear any previous error
				cookie.DeleteCookie(w, cookieName)
				continue
			}
			_, err := strconv.ParseFloat(val, 64)
			if err != nil && (isRequired || val != "") {
				errStr = validtr.api.Translate("error", "Input value must be a valid decimal number", "label", fieldLabel)
			} else if err == nil {
				valFloat, _ := strconv.ParseFloat(val, 64)
				errStr = validtr.validateDecimal(valFloat, minimum, maximum, fieldLabel)
			}

		case sdkapi.FormFieldTypeBoolean:
			formValues[fieldName] = val
			// No validation needed for boolean
			continue
		}

		if errStr != "" {
			cookie.SetCookie(w, cookieName, errStr)
			validateErr = errors.New("field validation error")
		} else {
			// Clear any previous error
			cookie.DeleteCookie(w, cookieName)
		}
	}

	if validateErr != nil {
		return nil, errors.New("form validation failed")
	}

	return &FormValues{values: formValues}, nil
}

// handleFileUpload handles file upload and returns the temp file path
func (validtr *HTTPFormValidator) handleFileUpload(r *http.Request, fieldName, fieldLabel string, isRequired bool, allowedExtensions string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			if isRequired {
				msg := validtr.api.Translate("error", "File upload is required", "label", fieldLabel)
				return "", fmt.Errorf(msg)
			}
			return "", nil
		}
		msg := validtr.api.Translate("error", "Failed to upload file", "label", fieldLabel)
		return "", fmt.Errorf(msg)
	}
	defer file.Close()

	// Validate file extension if specified
	if allowedExtensions != "" {
		ext := filepath.Ext(header.Filename)
		allowed := false
		for _, allowedExt := range strings.Split(allowedExtensions, ",") {
			if strings.EqualFold(ext, strings.TrimSpace(allowedExt)) {
				allowed = true
				break
			}
		}
		if !allowed {
			msg := validtr.api.Translate("error", "Invalid file extension uploaded", "label", fieldLabel, "extensions", allowedExtensions)
			return "", fmt.Errorf(msg)
		}
	}

	// Create temp directory for form uploads
	tmpDir := filepath.Join(sdkutils.PathTmpDir, "form-uploads", generateRandomID())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create temp file
	tmpFilePath := filepath.Join(tmpDir, header.Filename)
	out, err := os.Create(tmpFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// Stream file to temp location
	if _, err := io.Copy(out, file); err != nil {
		os.Remove(tmpFilePath)
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return tmpFilePath, nil
}

// generateRandomID generates a random ID for temp directories
func generateRandomID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
