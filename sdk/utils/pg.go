package sdkutils

import (
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

func PgUuidToString(uuid pgtype.UUID) string {
	return hex.EncodeToString(uuid.Bytes[:]) // Exclude nil bytes at the end if needed.
}

func PgStringToUUID(s string) pgtype.UUID {
	var uuid pgtype.UUID
	b, err := hex.DecodeString(s)
	if err != nil {
		fmt.Println("Error decoding string UUID:", err)
		return uuid // Return empty UUID if decoding failed.
	}
	copy(uuid.Bytes[:], b) // Copy decoded bytes into pgtype.UUID's Bytes.
	return uuid            // No errors occured, return filled out UUID.
}

func PgNumericToFloat64(numeric pgtype.Numeric) float64 {
	// Check for NaN
	if numeric.NaN {
		log.Println("numeric is NaN, returning 0")
		return float64(0)
	}

	if numeric.Int == nil {
		return float64(0)
	}

	if !numeric.Valid {
		return float64(0)
	}

	// Convert Int to *big.Float
	bigFloat := new(big.Float).SetInt(numeric.Int)

	// Apply the base-10 exponent
	scaleFactor := new(big.Float).SetFloat64(math.Pow10(int(numeric.Exp)))

	// Scale the value
	bigFloat.Mul(bigFloat, scaleFactor)

	// Convert to float64 (may lose precision for very large numbers)
	floatValue, _ := bigFloat.Float64()

	return floatValue
}

func PgFloat64ToNumeric(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric

	if err := numeric.Scan(fmt.Sprintf("%.2f", value)); err != nil {
		log.Println("Error converting float64 to pgtype.Numeric:", err)
		numeric.Valid = true
		return numeric // Return empty numeric if conversion failed.
	}
	numeric.Valid = true
	return numeric
}
