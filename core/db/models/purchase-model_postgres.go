//go:build postgres

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
// This is the PostgreSQL implementation using JSONB operators
func (self *PurchaseModel) FindPurchaseByAttrs(tx *sql.Tx, ctx context.Context, attrs map[string]string) (*Purchase, error) {
	if len(attrs) == 0 {
		return nil, fmt.Errorf("attrs map cannot be empty")
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	for key, value := range attrs {
		// Postgres JSONB syntax: metadata->>'key' = value
		conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT id, uid, device_id, sku, name, description, price, any_price, 
		       callback_plugin, callback_route, metadata, wallet_debit, wallet_tx_id,
		       confirmed_at, cancelled_at, cancelled_reason, created_at
		FROM purchases
		WHERE %s
		  AND confirmed_at IS NULL
		  AND cancelled_at IS NULL
		LIMIT 1
	`, strings.Join(conditions, " AND "))

	log.Printf("FindPurchaseByAttrs (Postgres) query: %s, args: %v", query, args)

	var qtx interface {
		QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	}

	if tx != nil {
		qtx = tx
	} else {
		qtx = self.db.DB
	}

	row := qtx.QueryRowContext(ctx, query, args...)

	var p queries.Purchase
	err := row.Scan(
		&p.ID,
		&p.Uid,
		&p.DeviceID,
		&p.Sku,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.AnyPrice,
		&p.CallbackPlugin,
		&p.CallbackRoute,
		&p.Metadata,
		&p.WalletDebit,
		&p.WalletTxID,
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
