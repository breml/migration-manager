package query

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/FuturFusion/migration-manager/internal/logger"
)

// Transaction executes the given function within a database transaction with a 30s context timeout.
func Transaction(ctx context.Context, db *sql.DB, f func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		// If there is a leftover transaction let's try to rollback,
		// we'll then retry again.
		if strings.Contains(err.Error(), "cannot start a transaction within a transaction") {
			_, _ = db.Exec("ROLLBACK")
		}

		return fmt.Errorf("Failed to begin transaction: %w", err)
	}

	err = f(ctx, tx)
	if err != nil {
		return rollback(tx, err)
	}

	err = tx.Commit()
	if err == sql.ErrTxDone {
		err = nil // Ignore duplicate commits/rollbacks
	}

	return err
}

// Rollback a transaction after the given error occurred. If the rollback
// succeeds the given error is returned, otherwise a new error that wraps it
// gets generated and returned.
func rollback(tx *sql.Tx, reason error) error {
	err := Retry(context.TODO(), func(_ context.Context) error { return tx.Rollback() })
	if err != nil {
		slog.Warn("Failed to rollback transaction after error", slog.Any("reason", reason), logger.Err(err))
	}

	return reason
}
