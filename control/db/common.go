package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *StateManager) Update(ctx context.Context, table string, updates map[string]interface{}, where_key string, where_value interface{}) error {
	fields := []string{}
	values := []interface{}{}
	paramCount := 1

	for field, value := range updates {
		fields = append(fields, field+" = $"+strconv.Itoa(paramCount))
		values = append(values, value)
		paramCount++
	}
	values = append(values, where_value)

	query := "UPDATE " + table + " SET " + strings.Join(fields, ", ") + " WHERE " + where_key + " = $" + strconv.Itoa(paramCount)

	return s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, values...)
		if err != nil {
			return fmt.Errorf("failed to execute update: %w", err)
		}
		return nil
	})
}

func (s *StateManager) Delete(ctx context.Context, table string, where_key string, where_value string) error {
	query := "DELETE FROM " + table + " WHERE " + where_key + " = $1"
	where_values := []interface{}{where_value}

	return s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, where_values...)
		if err != nil {
			return fmt.Errorf("failed to execute delete: %w", err)
		}
		return nil
	})
}
