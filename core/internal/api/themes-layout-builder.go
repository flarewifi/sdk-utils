package api

import (
	sdkapi "sdk/api"

	"github.com/a-h/templ"
)

type ThemesLayoutBuilder struct {
	FlashMessage   *sdkapi.FlashMsg
	PageContent    templ.Component
	ContentWrapper func(head, layout templ.Component)
}

// FlashMsg returns the flash message.
func (self *ThemesLayoutBuilder) FlashMsg() *sdkapi.FlashMsg {
	return self.FlashMessage
}

// Returns the page content
func (self *ThemesLayoutBuilder) Content() templ.Component {
	return self.PageContent
}

// Render the view
func (self *ThemesLayoutBuilder) Render(head templ.Component, layout templ.Component) {
	self.ContentWrapper(head, layout)
}
