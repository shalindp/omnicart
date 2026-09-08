package responses

import (
	"strings"
	"unicode"
)

// BarcodeType represents the namespace a barcode can identify a product in.
type BarcodeType int

const (
	BarcodeTypeNone     BarcodeType = iota
	BarcodeTypeGtin                 // 8, 12, 13, or 14 digits with valid check digit
	BarcodeTypePlu                  // 4 or 5 digits (produce codes)
	BarcodeTypeInternal             // other numeric codes
)

// BarcodeResult holds the classified barcode value and its type.
type BarcodeResult struct {
	Value string
	Type  BarcodeType
}

// gtinLengths are the valid GTIN lengths that Classify will pad to.
var gtinLengths = []int{8, 12, 13, 14}

// Classify determines the barcode type and returns the trimmed value.
// If a barcode value doesn't match a valid GTIN but is all digits, it pads
// with leading zeros to try each valid GTIN length (8, 12, 13, 14).
func Classify(raw string) BarcodeResult {
	value := strings.TrimSpace(raw)

	if len(value) == 0 || !isAllDigits(value) {
		return BarcodeResult{Value: "", Type: BarcodeTypeNone}
	}

	if len(value) < 4 || isAllZero(value) {
		return BarcodeResult{Value: "", Type: BarcodeTypeNone}
	}

	if IsValidGtin(value) {
		return BarcodeResult{Value: value, Type: BarcodeTypeGtin}
	}

	if len(value) == 4 || len(value) == 5 {
		return BarcodeResult{Value: value, Type: BarcodeTypePlu}
	}

	if padded, ok := PadToGtin(value); ok {
		return BarcodeResult{Value: padded, Type: BarcodeTypeGtin}
	}

	return BarcodeResult{Value: value, Type: BarcodeTypeInternal}
}

// PadToGtin left-pads a digit string with zeros to try each valid GTIN length.
// Returns the padded value and true if a valid GTIN is found.
func PadToGtin(value string) (string, bool) {
	for _, targetLength := range gtinLengths {
		if len(value) >= targetLength {
			continue
		}
		padded := strings.Repeat("0", targetLength-len(value)) + value
		if IsValidGtin(padded) {
			return padded, true
		}
	}
	return "", false
}

// IsValidGtin validates the GTIN check digit for 8, 12, 13, or 14-digit barcodes.
func IsValidGtin(candidate string) bool {
	if len(candidate) != 8 && len(candidate) != 12 && len(candidate) != 13 && len(candidate) != 14 {
		return false
	}

	if !isAllDigits(candidate) {
		return false
	}

	sum := 0
	weight := 3

	for index := len(candidate) - 2; index >= 0; index-- {
		sum += int(candidate[index]-'0') * weight
		if weight == 3 {
			weight = 1
		} else {
			weight = 3
		}
	}

	expectedCheckDigit := (10 - (sum % 10)) % 10
	return expectedCheckDigit == int(candidate[len(candidate)-1]-'0')
}

func isAllDigits(value string) bool {
	if len(value) == 0 {
		return false
	}
	for _, character := range value {
		if !unicode.IsDigit(character) {
			return false
		}
	}
	return true
}

func isAllZero(value string) bool {
	if len(value) == 0 {
		return false
	}
	for _, character := range value {
		if character != '0' {
			return false
		}
	}
	return true
}
