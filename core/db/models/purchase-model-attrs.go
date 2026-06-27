package models

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"core/db/queries"
)

// FindPurchaseByAttrs finds a pending purchase by matching metadata attributes
// This is the SQLite implementation using JSON functions
func (self *PurchaseModel) FindPurchaseByAttrs(ctx context.Context, attrs map[string]string) (*Purchase, error) {
	if len(attrs) == 0 {
		return nil, fmt.Errorf("attrs map cannot be empty")
	}

	// Build the WHERE clause for JSON matching in SQLite
	var conditions []string
	var args []interface{}

	for key, value := range attrs {
		// SQLite JSON syntax: json_extract(metadata, '$.key') = value
		conditions = append(conditions, fmt.Sprintf("json_extract(metadata, '$.%s') = ?", key))
		args = append(args, value)
	}

	// Build the complete query
	query := fmt.Sprintf(`
		SELECT id, uuid, device_id, sku, name, description, price, any_price,
		       callback_plugin, callback_route, metadata,
		       confirmed_at, cancelled_at, cancelled_reason, created_at
		FROM purchases
		WHERE %s
		  AND confirmed_at IS NULL
		  AND cancelled_at IS NULL
		LIMIT 1
	`, strings.Join(conditions, " AND "))

	// Execute the query
	row := self.db.DB.QueryRowContext(ctx, query, args...)

	var p queries.Purchase
	err := row.Scan(
		&p.ID,
		&p.Uuid,
		&p.DeviceID,
		&p.Sku,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.AnyPrice,
		&p.CallbackPlugin,
		&p.CallbackRoute,
		&p.Metadata,
		&p.ConfirmedAt,
		&p.CancelledAt,
		&p.CancelledReason,
		&p.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("purchase not found with attrs: %v", attrs)
		}
		log.Printf("error finding purchase by attrs %v: %v", attrs, err)
		return nil, err
	}

	return NewPurchase(self.db, self.models, &p)
}
