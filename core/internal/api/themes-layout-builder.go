package api

import (
	"github.com/a-h/templ"
)

type ThemesLayoutBuilder struct {
	PageContent    templ.Component
	ContentWrapper func(head, layout templ.Component)
}

// Returns the page content
func (self *ThemesLayoutBuilder) Content() templ.Component {
	return self.PageContent
}

// Render the view
func (self *ThemesLayoutBuilder) Render(head templ.Component, layout templ.Component) {
	self.ContentWrapper(head, layout)
}
