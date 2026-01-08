package browserdetect

import (
	"strings"

	"github.com/ua-parser/uap-go/uaparser"
)

type BrowserInfo struct {
	UserAgent   string
	IsCNA       bool
	OSFamily    string
	BrowserName string
}

var parser *uaparser.Parser

func init() {
	parser = uaparser.NewFromSaved()
}

// DetectBrowser analyzes User-Agent and returns browser information
func DetectBrowser(userAgent string) BrowserInfo {
	info := BrowserInfo{
		UserAgent: userAgent,
		IsCNA:     IsCaptivePortal(userAgent),
	}

	if userAgent == "" {
		return info
	}

	client := parser.Parse(userAgent)
	info.OSFamily = client.Os.Family
	info.BrowserName = client.UserAgent.Family

	return info
}

// IsCaptivePortal checks if User-Agent is from a captive portal browser
func IsCaptivePortal(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	return strings.Contains(ua, "captivenetworksupport") ||
		strings.Contains(ua, "wispr") ||
		strings.Contains(ua, "captiveportal") ||
		strings.Contains(ua, "ncsi")
}
