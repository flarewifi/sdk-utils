package pg

import (
	"log"
	"math"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

func NumericToFloat64(numeric pgtype.Numeric) float64 {
	// Check for NaN
	if numeric.NaN {
		log.Println("numeric is NaN, returning 0")
		return math.NaN()
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

func Float64ToNumeric(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric
	numeric.Scan(value)
	return numeric
}
