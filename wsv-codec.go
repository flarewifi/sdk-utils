package sdkutils

// WSV (Whitespace-Separated Values) line codec.
//
// A WSV line is a sequence of values separated by whitespace, where any value
// containing whitespace, a double quote, a hash, or a line break is wrapped in
// double quotes (with `""` escaping a literal quote and `"/"` escaping a line
// break). An empty value serializes to `""` and a literal `-` to `"-"`; a bare
// `-` decodes back to the empty string. A `#` outside of quotes starts a
// comment and ends the line.
//
// The core logger uses this to encode each log record's fields onto a single
// line and to parse those lines back when tailing. Only the two functions used
// there are exported: Serialize (encode) and ParseLineAsArray (decode).

import "errors"

const codepoint_LINEFEED = 0x0A
const codepoint_DOUBLEQUOTE = 0x22
const codepoint_HASH = 0x23
const codepoint_SLASH = 0x2F

type basicWsvCharIterator struct {
	chars []rune
	index int
}

// ParseLineAsArray parses a single WSV line into its field values.
func ParseLineAsArray(line string) ([]string, error) {
	return parseLine(line)
}

// Serialize encodes rows of values into a WSV document, one line per row.
func Serialize(rows [][]string) string {
	isFirst := true
	result := ""
	for _, row := range rows {
		if !isFirst {
			result += "\n"
		} else {
			isFirst = false
		}
		result += serializeRow(row)
	}
	return result
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

func (x *basicWsvCharIterator) isEnd() bool {
	return x.index >= len(x.chars)
}

func (x *basicWsvCharIterator) is(c rune) bool {
	return rune(x.chars[x.index]) == c
}

func (x *basicWsvCharIterator) isWhitespace() bool {
	return isWhitespace(rune(x.chars[x.index]))
}

func (x *basicWsvCharIterator) next() bool {
	x.index = x.index + 1
	return !x.isEnd()
}

func (x *basicWsvCharIterator) get() rune {
	return rune(x.chars[x.index])
}

func (x *basicWsvCharIterator) getSlice(startIndex int) []rune {
	return []rune(x.chars[startIndex:x.index])
}

func isWhitespace(c rune) bool {
	return c == 0x09 ||
		(c >= 0x0B && c <= 0x0D) ||
		c == 0x20 ||
		c == 0x85 ||
		c == 0xA0 ||
		c == 0x1680 ||
		(c >= 0x2000 && c <= 0x200A) ||
		(c >= 0x2028 && c <= 0x2029) ||
		c == 0x202F ||
		c == 0x205F ||
		c == 0x3000
}

func getCodePoints(s string) []rune {
	var result []rune
	for _, c := range s {
		result = append(result, rune(c))
	}
	return result
}

func parseLine(lineStrWithoutLinefeed string) ([]string, error) {
	var iterator = basicWsvCharIterator{getCodePoints(lineStrWithoutLinefeed), 0}
	var values []string
	for {
		skipWhitespace(&iterator)
		if iterator.isEnd() {
			break
		}
		if iterator.is(codepoint_HASH) {
			break
		}

		var curValue string
		if iterator.is(codepoint_DOUBLEQUOTE) {

			var err error
			curValue, err = parseDoubleQuotedValue(&iterator)
			if err != nil {
				return nil, err
			}
		} else {

			var err error
			curValue, err = parseValue(&iterator)
			if err != nil {
				return nil, err
			}
			if curValue == "-" {

				curValue = ""
			}
		}

		values = append(values, curValue)
	}

	return values, nil
}

func parseValue(iterator *basicWsvCharIterator) (string, error) {
	var startIndex = iterator.index
	for {
		if !iterator.next() {
			break
		}
		if iterator.isWhitespace() || iterator.is(codepoint_HASH) {
			break
		} else if iterator.is(codepoint_DOUBLEQUOTE) {
			return "", errors.New("invalid double quote in value")

		}
	}

	return string(iterator.getSlice(startIndex)), nil

}

func parseDoubleQuotedValue(iterator *basicWsvCharIterator) (string, error) {
	var value = ""
	for {
		if !iterator.next() {
			return "", errors.New("string not closed")
		}
		if iterator.is(codepoint_DOUBLEQUOTE) {
			if !iterator.next() {
				break
			}
			if iterator.is(codepoint_DOUBLEQUOTE) {
				value += "\""
			} else if iterator.is(codepoint_SLASH) {
				if !iterator.next() && iterator.is(codepoint_DOUBLEQUOTE) {
					return "", errors.New("invalid string line break")
				}
				value += "\n"
			} else if iterator.isWhitespace() || iterator.is(codepoint_HASH) {
				break
			} else {
				return "", errors.New("invalid character after string")
			}
		} else {
			value += string(iterator.get())
		}

	}
	return value, nil
}

func skipWhitespace(iterator *basicWsvCharIterator) {
	if iterator.isEnd() {
		return
	}

	//Bascially a do-while loop
	for ok := true; ok; ok = iterator.next() {
		if !iterator.isWhitespace() {
			break
		}
	}
}

func containsSpecialChar(chars []rune) bool {
	for _, c := range chars {
		if isWhitespace(c) || c == codepoint_LINEFEED || c == codepoint_DOUBLEQUOTE || c == codepoint_HASH {
			return true
		}
	}
	return false
}

func serializeValue(value string) string {
	if len(value) == 0 {
		return "\"\""
	} else if value == "-" {
		return "\"-\""
	} else {
		chars := getCodePoints(value)
		if containsSpecialChar(chars) {
			var result = "\""
			for _, c := range chars {
				switch c {
				case codepoint_LINEFEED:
					result += "\"/\""
				case codepoint_DOUBLEQUOTE:
					result += "\"\""
				default:
					result += string(c)
				}
			}
			result += "\""
			return result
		} else {
			return value
		}
	}
}

func serializeRow(values []string) string {
	isFirst := true
	result := ""
	for _, value := range values {
		if !isFirst {
			result += " "
		} else {
			isFirst = false
		}
		result += serializeValue(value)
	}
	return result
}
