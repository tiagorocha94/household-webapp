package service

import (
	"fmt"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

type goalSchedule struct {
	startMonth string
	endMonth   string
	schedule   goals.Schedule
	driftCents money.Cents // scheduled minus recorded; absorbed by the final month
}

// GetProjection builds a month-by-month projection for the given account:
// actual data from the movements for past months, and planned data for future
// months from active goals and the account's recurring deposit.
func (s *Service) GetProjection(accountID int64) ([]goals.ProjectionMonth, error) {
	account, err := s.GetAccount(accountID)
	if err != nil {
		return nil, err
	}

	endMonth, err := s.store.GetLastActiveGoalEndMonth(accountID)
	if err != nil {
		return nil, err
	}
	actuals, err := s.store.ListMonthlyActuals(accountID)
	if err != nil {
		return nil, err
	}
	plans, err := s.store.ListActiveGoalPlans(accountID)
	if err != nil {
		return nil, err
	}
	schedules, err := buildSchedules(plans)
	if err != nil {
		return nil, err
	}
	return buildProjection(time.Now(), endMonth, account.MonthlyDepositCents, actuals, schedules)
}

func buildSchedules(plans []goals.GoalPlan) ([]goalSchedule, error) {
	schedules := make([]goalSchedule, 0, len(plans))
	for _, p := range plans {
		schedule, err := goals.NewSchedule(p.StartMonth, p.EndMonth, p.TargetCents)
		if err != nil {
			return nil, fmt.Errorf("size goal schedule: %w", err)
		}
		schedules = append(schedules, goalSchedule{
			startMonth: p.StartMonth,
			endMonth:   p.EndMonth,
			schedule:   schedule,
			driftCents: schedule.Drift(p.AllocationCount, p.AllocatedCents),
		})
	}
	return schedules, nil
}

// buildProjection covers 12 months back to endMonth (or six months ahead when
// endMonth is empty).
func buildProjection(now time.Time, endMonth string, monthlyDeposit money.Cents, actuals map[string]*goals.MonthActual, schedules []goalSchedule) ([]goals.ProjectionMonth, error) {
	startMonth := month.Offset(now, -12)
	if endMonth == "" {
		endMonth = month.Offset(now, 6)
	}
	currentMonth := month.Current(now)

	window, err := month.Range(startMonth, endMonth)
	if err != nil {
		return nil, fmt.Errorf("list projection months: %w", err)
	}

	var projection []goals.ProjectionMonth
	for _, m := range window {
		pm := goals.ProjectionMonth{Month: m, IsProjection: m > currentMonth}

		if pm.IsProjection {
			pm.DepositsCents = monthlyDeposit
			pm.AllocationsCents = getPlannedAllocations(schedules, m)
		} else if actual, ok := actuals[m]; ok {
			pm.DepositsCents = actual.DepositsCents
			pm.AllocationsCents = actual.AllocationsCents
			pm.ExpensesCents = actual.ExpensesCents
			pm.ReleasedCents = actual.ReleasedCents
			pm.CoveredCents = actual.CoveredCents
		}
		projection = append(projection, pm)
	}

	// Running totals start from everything recorded before the window.
	running := sumBefore(actuals, startMonth)
	for i := range projection {
		running.DepositsCents += projection[i].DepositsCents
		running.AllocationsCents += projection[i].AllocationsCents
		running.ExpensesCents += projection[i].ExpensesCents
		running.ReleasedCents += projection[i].ReleasedCents
		running.CoveredCents += projection[i].CoveredCents
		projection[i].BalanceCents = running.DepositsCents - running.ExpensesCents
		projection[i].UnallocatedCents = running.DepositsCents - running.AllocationsCents + running.ReleasedCents - running.CoveredCents
	}
	return projection, nil
}

// sumBefore totals every actual strictly before startMonth.
func sumBefore(actuals map[string]*goals.MonthActual, startMonth string) goals.MonthActual {
	var total goals.MonthActual
	for m, a := range actuals {
		if m < startMonth {
			total.DepositsCents += a.DepositsCents
			total.AllocationsCents += a.AllocationsCents
			total.ExpensesCents += a.ExpensesCents
			total.ReleasedCents += a.ReleasedCents
			total.CoveredCents += a.CoveredCents
		}
	}
	return total
}

func getPlannedAllocations(schedules []goalSchedule, m string) money.Cents {
	var total money.Cents
	for _, g := range schedules {
		if m < g.startMonth || m > g.endMonth {
			continue
		}
		position, err := month.Count(g.startMonth, m)
		if err != nil {
			continue
		}
		total += max(g.schedule.Amount(position-1, g.driftCents), 0)
	}
	return total
}
