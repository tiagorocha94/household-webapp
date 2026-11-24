package goals

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// syncGoals runs the goals module's automated maintenance as of now:
//  1. Inserts missing allocation movements for active goals up to the current month.
//  2. Settles the goals that were completed by hand: closing movement and
//     closing balance. Goals never complete on their own.
func syncGoals(db *sql.DB, now time.Time, log *slog.Logger) error {
	currentMonth := month.Current(now)

	if err := syncAllocations(db, currentMonth, log); err != nil {
		return fmt.Errorf("sync allocations: %w", err)
	}

	if err := syncClosings(db, currentMonth, log); err != nil {
		return fmt.Errorf("sync closings: %w", err)
	}

	return nil
}

func syncAllocations(db *sql.DB, currentMonth string, log *slog.Logger) error {
	rows, err := db.Query(
		`SELECT g.id, g.account_id, g.start_month, g.end_month, g.target_cents
		 FROM goal_account_goals g
		 WHERE g.status = 'active'`,
	)
	if err != nil {
		return fmt.Errorf("query active goals: %w", err)
	}
	defer rows.Close()

	type goalInfo struct {
		id        int64
		accountID int64
		start     string
		end       string
		target    money.Cents
		schedule  Schedule
	}

	var active []goalInfo
	for rows.Next() {
		var g goalInfo
		if err := rows.Scan(&g.id, &g.accountID, &g.start, &g.end, &g.target); err != nil {
			return fmt.Errorf("scan goal: %w", err)
		}
		g.schedule, err = NewSchedule(g.start, g.end, g.target)
		if err != nil {
			return fmt.Errorf("schedule goal %d: %w", g.id, err)
		}
		active = append(active, g)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, g := range active {
		due, err := month.Range(g.start, min(currentMonth, g.end))
		if err != nil {
			return fmt.Errorf("list due months of goal %d: %w", g.id, err)
		}

		count := 0
		for i, m := range due {
			if i >= g.schedule.Months() {
				break
			}
			amount := g.schedule.Amount(i, 0)
			if i == g.schedule.Months()-1 {
				// The final month absorbs rounding drift from earlier months.
				var others money.Cents
				if err := db.QueryRow(
					`SELECT COALESCE(SUM(amount_cents), 0) FROM goal_account_movements
					 WHERE goal_id = ? AND type = 'allocation' AND month <> ?`,
					g.id, m,
				).Scan(&others); err != nil {
					return fmt.Errorf("sum allocations for goal %d: %w", g.id, err)
				}
				amount = g.schedule.FinalAmount(others)
				if amount <= 0 {
					// Over-allocated: nothing left to allocate.
					continue
				}
			}
			result, err := db.Exec(
				`INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month)
				 SELECT ?, ?, 'allocation', ?, ?
				 WHERE NOT EXISTS (
				   SELECT 1 FROM goal_account_movements
				   WHERE goal_id = ? AND type = 'allocation' AND month = ?
				 )`,
				g.accountID, g.id, amount, m, g.id, m,
			)
			if err != nil {
				return fmt.Errorf("insert allocation for goal %d month %s: %w", g.id, m, err)
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("count allocation inserted for goal %d month %s: %w", g.id, m, err)
			}
			count += int(affected)
		}

		if count > 0 {
			log.Info("auto-allocated goal months", "goal_id", g.id, "months", count)
		}
	}

	return nil
}

// syncClosings settles the goals that were completed by hand. Completing is a
// status change in the goal CSV; nothing completes on its own, because the
// expenses of a goal often come after its last allocation.
//
// A completed goal without a closing balance gets one: its leftover returns to
// the account's unallocated money as a release, or the account covers what it
// overspent, and the balance it had is kept on the goal. That balance matches
// the closing movement. A goal that is not completed has no closing balance.
func syncClosings(db *sql.DB, currentMonth string, log *slog.Logger) error {
	if _, err := db.Exec(
		`UPDATE goal_account_goals SET closing_balance_cents = NULL
		 WHERE status <> 'completed' AND closing_balance_cents IS NOT NULL`,
	); err != nil {
		return fmt.Errorf("clear closing balances of goals that are not completed: %w", err)
	}

	rows, err := db.Query(
		`SELECT id, account_id FROM goal_account_goals
		 WHERE status = 'completed' AND closing_balance_cents IS NULL`,
	)
	if err != nil {
		return fmt.Errorf("query completed goals to settle: %w", err)
	}
	defer rows.Close()

	type pending struct{ id, accountID int64 }
	var goalsToSettle []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.accountID); err != nil {
			return fmt.Errorf("scan completed goal: %w", err)
		}
		goalsToSettle = append(goalsToSettle, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, g := range goalsToSettle {
		if err := settleGoal(db, g.accountID, g.id, currentMonth, log); err != nil {
			return err
		}
	}
	return nil
}

// settleGoal records the closing movement of a completed goal, if it has none
// yet, and sets its closing balance, in one transaction so a failure cannot
// leave a settled goal without its balance (the next sync would settle it twice).
func settleGoal(db *sql.DB, accountID, goalID int64, month string, log *slog.Logger) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin settling goal %d: %w", goalID, err)
	}
	defer tx.Rollback()

	var closings int
	var closing money.Cents
	if err := tx.QueryRow(
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN type = 'release' THEN amount_cents ELSE -amount_cents END), 0)
		 FROM goal_account_movements WHERE goal_id = ? AND type IN ('release', 'cover')`,
		goalID,
	).Scan(&closings, &closing); err != nil {
		return fmt.Errorf("read closing movements of goal %d: %w", goalID, err)
	}

	if closings == 0 {
		// The balance before any closing movement: allocated less spent.
		if err := tx.QueryRow(
			`SELECT COALESCE(SUM(CASE WHEN type = 'allocation' THEN amount_cents
			                          WHEN type = 'expense' THEN -amount_cents ELSE 0 END), 0)
			 FROM goal_account_movements WHERE goal_id = ?`,
			goalID,
		).Scan(&closing); err != nil {
			return fmt.Errorf("compute balance of goal %d: %w", goalID, err)
		}
		movType, amount, description := MovementRelease, closing, "Goal completed: leftover returned to the account"
		if closing < 0 {
			movType, amount, description = MovementCover, -closing, "Goal completed: overspend covered by the account"
		}
		if closing != 0 {
			if _, err := tx.Exec(
				`INSERT INTO goal_account_movements (account_id, goal_id, type, amount_cents, month, description)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				accountID, goalID, movType, amount, month, description,
			); err != nil {
				return fmt.Errorf("insert %s for goal %d: %w", movType, goalID, err)
			}
		}
	}

	if _, err := tx.Exec(
		"UPDATE goal_account_goals SET closing_balance_cents = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		closing, goalID,
	); err != nil {
		return fmt.Errorf("set closing balance of goal %d: %w", goalID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit settling goal %d: %w", goalID, err)
	}
	log.Info("settled completed goal", "goal_id", goalID, "closing_balance_cents", int64(closing))
	return nil
}
