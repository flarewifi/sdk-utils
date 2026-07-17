package sdkutils

import (
	"strconv"
	"strings"
)

// List of supported currencies
const (
	CurrencyPhilippinePeso string = "PHP"
	CurrencyUsDollar       string = "USD"
	CurrencyNigerianNaira  string = "NGN"
)

// SupportedCurrency represents a supported currency with its details
type SupportedCurrency struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

// SupportedCurrencies represents the list of supported currencies
var SupportedCurrencies = []SupportedCurrency{
	{Code: CurrencyUsDollar, Name: "US Dollar", Symbol: "$"},
	{Code: CurrencyPhilippinePeso, Name: "Philippine Peso", Symbol: "₱"},
	{Code: CurrencyNigerianNaira, Name: "Nigerian Naira", Symbol: "₦"},
}

// IsValidCurrency checks if the given currency code is supported
func IsValidCurrency(currencyCode string) bool {
	for _, currency := range SupportedCurrencies {
		if currency.Code == currencyCode {
			return true
		}
	}
	return false
}

// GetCurrencyByName returns currency information by name
func GetCurrencyByName(name string) (SupportedCurrency, bool) {
	for _, currency := range SupportedCurrencies {
		if strings.EqualFold(currency.Name, name) {
			return currency, true
		}
	}
	return SupportedCurrency{}, false
}

// GetCurrencyByCode returns currency information by code
func GetCurrencyByCode(code string) (SupportedCurrency, bool) {
	for _, currency := range SupportedCurrencies {
		if currency.Code == code {
			return currency, true
		}
	}
	return SupportedCurrency{}, false
}

// currencyDisplaySymbols maps ISO 4217 codes to their display symbol for
// currencies the machine does not itself transact in but may need to *render*
// (e.g. a store price shown in the owner's country currency). This is kept
// separate from SupportedCurrencies on purpose: SupportedCurrencies is the set
// the machine operates in (drives IsValidCurrency / ParseCurrencyAmount), while
// this table is display-only. Covers the major world currencies; anything not
// listed still falls back to the bare code, which is acceptable.
var currencyDisplaySymbols = map[string]string{
	"USD": "$", "PHP": "₱", "NGN": "₦",
	"EUR": "€", "GBP": "£", "JPY": "¥", "CNY": "¥", "INR": "₹",
	"AUD": "A$", "CAD": "C$", "NZD": "NZ$", "SGD": "S$", "HKD": "HK$",
	"CHF": "CHF", "SEK": "kr", "NOK": "kr", "DKK": "kr", "PLN": "zł",
	"RUB": "₽", "TRY": "₺", "BRL": "R$", "MXN": "Mex$", "ZAR": "R",
	"KRW": "₩", "THB": "฿", "IDR": "Rp", "MYR": "RM", "VND": "₫",
	"AED": "د.إ", "SAR": "﷼", "ILS": "₪", "EGP": "E£", "KES": "KSh",
	"GHS": "₵", "UAH": "₴", "CZK": "Kč", "HUF": "Ft", "RON": "lei",
	"TWD": "NT$", "PKR": "₨", "BDT": "৳", "LKR": "Rs", "ARS": "$",
	"CLP": "$", "COP": "$", "PEN": "S/", "BHD": ".د.ب", "QAR": "﷼",
}

// GetCurrencySymbol returns the currency symbol for the given currency code.
// The machine's operating currencies (SupportedCurrencies) win first so their
// curated symbols stay authoritative; otherwise a broader display table is
// consulted so store prices in a buyer's own currency render with the right
// glyph instead of a bare ISO code. Unknown codes fall back to the code itself.
func GetCurrencySymbol(currencyCode string) string {
	if currency, exists := GetCurrencyByCode(currencyCode); exists {
		return currency.Symbol
	}
	if symbol, exists := currencyDisplaySymbols[strings.ToUpper(strings.TrimSpace(currencyCode))]; exists {
		return symbol
	}
	return currencyCode // Fallback to currency code if not found
}

// ParseCurrencyAmount parses a currency string and returns the numeric amount
func ParseCurrencyAmount(currencyStr string) (float64, error) {
	// Remove all currency symbols from the supported currencies table
	for _, currency := range SupportedCurrencies {
		currencyStr = strings.ReplaceAll(currencyStr, currency.Symbol, "")
	}

	// Remove commas and whitespace
	currencyStr = strings.ReplaceAll(currencyStr, ",", "")
	currencyStr = strings.TrimSpace(currencyStr)

	return strconv.ParseFloat(currencyStr, 64)
}
