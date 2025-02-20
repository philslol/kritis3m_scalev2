package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (s *StateManager) Update(ctx context.Context, table string, updates map[string]interface{}, where_key string, where_value string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	fields := []string{}
	values := []interface{}{}

	paramCount := 1

	for field, value := range updates {
		fields = append(fields, field+" = $"+string(paramCount+'0'))
		values = append(values, value)
		paramCount++
	}
	//we dont need to increment paramCount, since ++ is post-increment
	values = append(values, where_value)
	
	query := "UPDATE " + table + " SET " + strings.Join(fields, ", ") + " WHERE " + where_key + " = $" + string(paramCount+'0')

	_, err = s.pool.Exec(ctx, query, values...)

	return tx.Commit(ctx)
}


func (s *StateManager) Delete(ctx context.Context, table string, where_key string, value string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := "DELETE FROM " + table + " WHERE " + where_key + " = $1")

	_, err = s.pool.Exec(ctx, query, value...)

	if err != nil{
		return fmt.Errorf("failed to commit transaction: %w", err
	}


	return tx.Commit(ctx)

}
