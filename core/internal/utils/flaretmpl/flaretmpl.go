package flaretmpl

import (
	htmltemplate "html/template"
	"os"
	"sync"
	texttemplate "text/template"

	"tools/env"
)

var (
	htmlTmplCache = sync.Map{}
	textTmplCache = sync.Map{}
	useCache      = env.GO_ENV != env.ENV_DEV
)

func GetHtmlTemplate(path string) (*htmltemplate.Template, error) {
	if v, ok := htmlTmplCache.Load(path); ok && useCache {
		return v.(*htmltemplate.Template), nil
	}

	tmplContent, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tmpl, err := htmltemplate.New(path).Delims("<%", "%>").Parse(string(tmplContent))
	if err != nil {
		return nil, err
	}

	htmlTmplCache.Store(path, tmpl)

	return tmpl, nil
}

func GetTextTemplate(path string) (*texttemplate.Template, error) {
	if v, ok := textTmplCache.Load(path); ok && useCache {
		return v.(*texttemplate.Template), nil
	}

	tmplContent, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tmpl, err := texttemplate.New(path).Delims("<%", "%>").Parse(string(tmplContent))
	if err != nil {
		return nil, err
	}

	textTmplCache.Store(path, tmpl)
	return tmpl, nil
}

// ClearCache clears all cached templates
// This should be called when switching languages to ensure new translations are loaded
func ClearCache() {
	htmlTmplCache.Range(func(key, value interface{}) bool {
		htmlTmplCache.Delete(key)
		return true
	})

	textTmplCache.Range(func(key, value interface{}) bool {
		textTmplCache.Delete(key)
		return true
	})
}
