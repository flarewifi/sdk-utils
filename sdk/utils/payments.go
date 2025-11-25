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

// GetCurrencySymbol returns the currency symbol for the given currency code
func GetCurrencySymbol(currencyCode string) string {
	if currency, exists := GetCurrencyByCode(currencyCode); exists {
		return currency.Symbol
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
