package api

import (
	"github.com/a-h/templ"
)

type ThemesLayoutBuilder struct {
	headContent    templ.Component
	pageContent    templ.Component
	scriptsContent templ.Component
	htmlAttrs      templ.Attributes
	bodyAttrs      templ.Attributes
}

func (self *ThemesLayoutBuilder) HtmlAttrs() templ.Attributes {
	return self.htmlAttrs
}

// Returns the page content
func (self *ThemesLayoutBuilder) Head() templ.Component {
	return self.headContent
}

func (self *ThemesLayoutBuilder) BodyAttrs() templ.Attributes {
	return self.bodyAttrs
}

func (self *ThemesLayoutBuilder) PageContent() templ.Component {
	return self.pageContent
}

func (self *ThemesLayoutBuilder) Scripts() templ.Component {
	return self.scriptsContent
}
