package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func (s *StateManager) Update(ctx context.Context, table string, updates map[string]interface{}, where_key string, where_value interface{}) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	fields := []string{}
	values := []interface{}{}

	paramCount := 1

	for field, value := range updates {
		fields = append(fields, field+" = $"+strconv.Itoa(paramCount))
		values = append(values, value)
		paramCount++
	}
	//we dont need to increment paramCount, since ++ is post-increment
	values = append(values, where_value)

	query := "UPDATE " + table + " SET " + strings.Join(fields, ", ") + " WHERE " + where_key + " = $" + strconv.Itoa(paramCount)

	_, err = s.pool.Exec(ctx, query, values...)

	return tx.Commit(ctx)
}

func (s *StateManager) Delete(ctx context.Context, table string, where_key string, where_value string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := "DELETE FROM " + table + " WHERE " + where_key + " = $1"
	where_values := []interface{}{where_value}

	_, err = s.pool.Exec(ctx, query, where_values...)

	return tx.Commit(ctx)

}
