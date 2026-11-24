package store

import (
	"fmt"

	"github.com/tiagorocha94/household-webapp/internal/cashflow"
)

// ListCategories returns every category, ordered by name.
func (s *Store) ListCategories() ([]cashflow.Category, error) {
	rows, err := s.db.Query(
		`SELECT id, name, direction, icon, color, COALESCE(role, ''), created_at, updated_at
		 FROM cashflow_categories ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list cashflow categories: %w", err)
	}
	defer rows.Close()

	var categories []cashflow.Category
	for rows.Next() {
		var c cashflow.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Direction, &c.Icon, &c.Color, &c.Role, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan cashflow category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// ListBudgetGroups returns the budget groups in the order of the seed file.
func (s *Store) ListBudgetGroups() ([]cashflow.BudgetGroup, error) {
	rows, err := s.db.Query(
		`SELECT id, name, share_pct FROM cashflow_budget_groups ORDER BY position ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list cashflow budget groups: %w", err)
	}
	defer rows.Close()

	var groups []cashflow.BudgetGroup
	for rows.Next() {
		var g cashflow.BudgetGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.SharePct); err != nil {
			return nil, fmt.Errorf("scan cashflow budget group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// ListCategoryGroups returns the budget group id of every category that has
// one, keyed by category id.
func (s *Store) ListCategoryGroups() (map[int64]int64, error) {
	rows, err := s.db.Query(
		`SELECT id, budget_group_id FROM cashflow_categories WHERE budget_group_id IS NOT NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("list cashflow category groups: %w", err)
	}
	defer rows.Close()

	groups := map[int64]int64{}
	for rows.Next() {
		var categoryID, groupID int64
		if err := rows.Scan(&categoryID, &groupID); err != nil {
			return nil, fmt.Errorf("scan cashflow category group: %w", err)
		}
		groups[categoryID] = groupID
	}
	return groups, rows.Err()
}
