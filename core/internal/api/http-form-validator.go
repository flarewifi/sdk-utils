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

	sdkutils "github.com/flarewifi/sdk-utils"
)

func NewHTTPFormValidator(api *PluginApi) *HTTPFormValidator {
	return &HTTPFormValidator{
		api: api,
	}
}

type HTTPFormValidator struct {
	api *PluginApi
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

func (validtr *HTTPFormValidator) validateDecimal(val any, min, max float64, label string) (errStr string) {
	fval := val.(float64)
	if fval < min {
		errStr = validtr.api.Translate("error", "Input value does not meet the required minimum", "label", label, "min", min)
	}

	if max != 0 && fval > max {
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
		isRequired := rules.Required
		allowedExtensions := rules.FileExt

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
			min := 0
			max := 0
			if rules.Minimum != "" {
				min, _ = strconv.Atoi(rules.Minimum)
			}
			if rules.Maximum != "" {
				max, _ = strconv.Atoi(rules.Maximum)
			}
			formValues[fieldName] = val
			errStr = validtr.validateString(isRequired, val, fieldLabel, min, max)
			if rules.Email {
				if !isValidEmail(val) {
					errStr = validtr.api.Translate("error", "Input value must be a valid email", "label", fieldLabel)
				}
			}
			if rules.Number {
				if _, err := strconv.ParseFloat(val, 64); err != nil {
					errStr = validtr.api.Translate("error", "Input value must be a number", "label", fieldLabel)
				}
			}

		case sdkapi.FormFieldTypeInteger:
			min := 0
			max := 0
			if rules.Minimum != "" {
				min, _ = strconv.Atoi(rules.Minimum)
			}
			if rules.Maximum != "" {
				max, _ = strconv.Atoi(rules.Maximum)
			}
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
				errStr = validtr.validateInteger(valInt, min, max, fieldLabel)
			}

		case sdkapi.FormFieldTypeDecimal:
			minF := 0.0
			maxF := 0.0
			if rules.Minimum != "" {
				minF, _ = strconv.ParseFloat(rules.Minimum, 64)
			}
			if rules.Maximum != "" {
				maxF, _ = strconv.ParseFloat(rules.Maximum, 64)
			}
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
				errStr = validtr.validateDecimal(valFloat, minF, maxF, fieldLabel)
			}

		case sdkapi.FormFieldTypeBoolean:
			formValues[fieldName] = val
			// No validation needed for boolean
			continue
		}

		if errStr != "" {
			cookie.SetCookie(w, cookieName, errStr, nil)
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

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Simple email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]
	if local == "" || domain == "" {
		return false
	}
	if !strings.Contains(domain, ".") {
		return false
	}
	return true
}

// generateRandomID generates a random ID for temp directories
func generateRandomID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
