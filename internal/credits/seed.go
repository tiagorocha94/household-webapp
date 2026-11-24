package credits

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvdata"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/money"
	"github.com/tiagorocha94/household-webapp/internal/month"
)

// moduleName names the module in logs, errors and the csv state.
const moduleName = "credits"

// SeedModule imports and exports the credit CSV files and satisfies the seed
// package's Module interface.
type SeedModule struct{}

type creditContractSeed struct {
	name                   string
	category               string
	contractNumber         string
	lender                 string
	originalPrincipalCents money.Cents
	startMonth             string
	expectedEndMonth       string
	notes                  string
	expectedDay            int
}

type creditPaymentSeed struct {
	contractID     int64
	month          string
	paymentType    string
	principalCents money.Cents
	interestCents  money.Cents
	description    string
	movement       string // sync_id of the cashflow movement it belongs to, or empty
}

type creditContractLookup struct {
	startMonth             string
	originalPrincipalCents money.Cents
}

// Name returns the module's name for logs and errors.
func (SeedModule) Name() string { return moduleName }

// Files lists the CSV files the module imports, with their headers.
func (SeedModule) Files() []csvdata.File {
	return []csvdata.File{
		{Name: "credit_contracts.csv", Header: []string{"name", "category", "contract_number", "lender", "original_principal_cents", "start_month", "expected_end_month", "notes", "expected_day"}},
		{Name: "credit_payments.csv", Header: []string{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"}},
	}
}

// Import mirrors the credit CSV files that exist into the database.
func (SeedModule) Import(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error {
	return applyCredits(tx, files, log)
}

// AfterExport records that the files now hold everything in the database. The
// seed runner calls it once they are written, at start-up, before a device can
// change anything.
func (SeedModule) AfterExport(db *sql.DB) error {
	return csvstate.MarkWritten(db, moduleName)
}

// Sync does nothing: credit status is derived by a view, not maintained.
func (SeedModule) Sync(*sql.DB, time.Time, *slog.Logger) error { return nil }

// Export returns the credit tables in write order.
func (SeedModule) Export(db *sql.DB) ([]csvdata.Table, error) {
	steps := []struct {
		file string
		fn   func(*sql.DB) ([][]string, error)
	}{
		{"credit_contracts.csv", exportCreditContracts},
		{"credit_payments.csv", exportCreditPayments},
	}
	tables := make([]csvdata.Table, 0, len(steps))
	for _, step := range steps {
		rows, err := step.fn(db)
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", step.file, err)
		}
		tables = append(tables, csvdata.Table{Name: step.file, Rows: rows})
	}
	return tables, nil
}

func exportCreditContracts(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT name, category, contract_number, lender,
		        original_principal_cents, start_month, COALESCE(expected_end_month, ''),
		        COALESCE(notes, ''), COALESCE(expected_day, 0)
		 FROM credit_contracts
		 ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query credit contracts for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{{"name", "category", "contract_number", "lender", "original_principal_cents", "start_month", "expected_end_month", "notes", "expected_day"}}
	for rows.Next() {
		var name, category, number, lender, start, end, notes string
		var principal money.Cents
		var day int
		if err := rows.Scan(&name, &category, &number, &lender, &principal, &start, &end, &notes, &day); err != nil {
			return nil, fmt.Errorf("scan credit contract for export: %w", err)
		}
		result = append(result, []string{name, category, number, lender, csvdata.FormatCents(principal), start, end, notes, formatDay(day)})
	}
	return result, rows.Err()
}

func exportCreditPayments(db *sql.DB) ([][]string, error) {
	rows, err := db.Query(
		`SELECT c.name, p.month, p.type, p.principal_cents,
		        p.interest_cents, COALESCE(p.description, ''), COALESCE(p.movement_sync_id, '')
		 FROM credit_payments p
		 JOIN credit_contracts c ON c.id = p.contract_id
		 ORDER BY c.name ASC, p.month ASC, p.type ASC, p.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query credit payments for export: %w", err)
	}
	defer rows.Close()

	result := [][]string{{"credit", "month", "type", "principal_cents", "interest_cents", "description", "movement"}}
	for rows.Next() {
		var credit, month, paymentType, description, movement string
		var principal, interest money.Cents
		if err := rows.Scan(&credit, &month, &paymentType, &principal, &interest, &description, &movement); err != nil {
			return nil, fmt.Errorf("scan credit payment for export: %w", err)
		}
		result = append(result, []string{
			credit, month, paymentType,
			csvdata.FormatCents(principal), csvdata.FormatCents(interest), description, movement,
		})
	}
	return result, rows.Err()
}

func getCreditContract(q database.Querier, name string) (int64, creditContractLookup, error) {
	var id int64
	var contract creditContractLookup
	err := q.QueryRow(
		"SELECT id, start_month, original_principal_cents FROM credit_contracts WHERE name = ?",
		name,
	).Scan(&id, &contract.startMonth, &contract.originalPrincipalCents)
	if err != nil {
		return 0, creditContractLookup{}, fmt.Errorf("credit %q not found: %w", name, err)
	}
	return id, contract, nil
}

func validateCreditContractSeed(contract creditContractSeed) error {
	if contract.name == "" || contract.contractNumber == "" || contract.lender == "" {
		return errors.New("name, contract_number, and lender are required")
	}
	switch contract.category {
	case "mortgage", "car", "personal", "other":
	default:
		return fmt.Errorf("invalid category %q", contract.category)
	}
	if err := month.Validate(contract.startMonth); err != nil {
		return fmt.Errorf("invalid start_month: %w", err)
	}
	if contract.expectedEndMonth != "" {
		if err := month.Validate(contract.expectedEndMonth); err != nil {
			return fmt.Errorf("invalid expected_end_month: %w", err)
		}
		if contract.expectedEndMonth < contract.startMonth {
			return errors.New("expected_end_month cannot be before start_month")
		}
	}
	if contract.expectedDay != 0 && (contract.expectedDay < 1 || contract.expectedDay > 28) {
		return fmt.Errorf("invalid expected_day %d: expected 1 to 28 or empty", contract.expectedDay)
	}
	return nil
}

// formatDay writes an expected day for the CSV; no day is empty.
func formatDay(day int) string {
	if day == 0 {
		return ""
	}
	return strconv.Itoa(day)
}

func validateCreditPaymentSeed(contract creditContractLookup, payment creditPaymentSeed) error {
	if err := month.Validate(payment.month); err != nil {
		return fmt.Errorf("invalid month: %w", err)
	}
	if payment.month < contract.startMonth {
		return errors.New("payment month cannot be before credit start_month")
	}
	if payment.paymentType != "regular" && payment.paymentType != "extraordinary" {
		return fmt.Errorf("invalid payment type %q", payment.paymentType)
	}
	if payment.principalCents+payment.interestCents <= 0 {
		return errors.New("payment total must be positive")
	}
	return nil
}

func applyCredits(tx *sql.Tx, files map[string][][]string, log *slog.Logger) error {
	if rows, ok := files["credit_contracts.csv"]; ok {
		if err := mirrorCreditContracts(tx, rows, log); err != nil {
			return fmt.Errorf("credit_contracts.csv: %w", err)
		}
	}
	// Devices write payments. When they changed the database after the files
	// were last written, the files are older and importing them would undo
	// that; the export that follows rewrites them.
	behind, err := csvstate.Behind(tx, moduleName)
	if err != nil {
		return err
	}
	if behind {
		log.Warn("not importing credit payments: the database has changes the files do not have yet")
		return nil
	}
	if rows, ok := files["credit_payments.csv"]; ok {
		if err := mirrorCreditPayments(tx, rows); err != nil {
			return fmt.Errorf("credit_payments.csv: %w", err)
		}
	}
	return nil
}

func mirrorCreditContracts(q database.Querier, rows [][]string, log *slog.Logger) error {
	contracts := make([]creditContractSeed, 0, len(rows))
	for i, row := range rows {
		if len(row) < 7 {
			return fmt.Errorf("row %d: expected at least 7 columns", i+2)
		}
		principal, err := csvdata.ParseNonNegativeCents(row[4], "original_principal_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		if principal <= 0 {
			return fmt.Errorf("row %d: original_principal_cents must be positive", i+2)
		}
		contract := creditContractSeed{
			name:                   strings.TrimSpace(row[0]),
			category:               strings.TrimSpace(row[1]),
			contractNumber:         strings.TrimSpace(row[2]),
			lender:                 strings.TrimSpace(row[3]),
			originalPrincipalCents: principal,
			startMonth:             strings.TrimSpace(row[5]),
			expectedEndMonth:       strings.TrimSpace(row[6]),
		}
		if len(row) >= 8 {
			contract.notes = strings.TrimSpace(row[7])
		}
		if len(row) >= 9 && strings.TrimSpace(row[8]) != "" {
			day, err := strconv.Atoi(strings.TrimSpace(row[8]))
			if err != nil || day < 1 {
				return fmt.Errorf("row %d: invalid expected_day %q: expected 1 to 28 or empty", i+2, row[8])
			}
			contract.expectedDay = day
		}
		if err := validateCreditContractSeed(contract); err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		contracts = append(contracts, contract)
	}

	keep := make(map[string]bool, len(contracts))
	for _, c := range contracts {
		// The update only fires when a column changed.
		if _, err := q.Exec(
			`INSERT INTO credit_contracts
			 (name, category, contract_number, lender, original_principal_cents,
			  start_month, expected_end_month, notes, expected_day)
			 VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, 0))
			 ON CONFLICT(name) DO UPDATE SET
			   category = excluded.category,
			   contract_number = excluded.contract_number,
			   lender = excluded.lender,
			   original_principal_cents = excluded.original_principal_cents,
			   start_month = excluded.start_month,
			   expected_end_month = excluded.expected_end_month,
			   notes = excluded.notes,
			   expected_day = excluded.expected_day,
			   updated_at = CURRENT_TIMESTAMP
			 WHERE credit_contracts.category IS NOT excluded.category
			    OR credit_contracts.contract_number IS NOT excluded.contract_number
			    OR credit_contracts.lender IS NOT excluded.lender
			    OR credit_contracts.original_principal_cents IS NOT excluded.original_principal_cents
			    OR credit_contracts.start_month IS NOT excluded.start_month
			    OR credit_contracts.expected_end_month IS NOT excluded.expected_end_month
			    OR credit_contracts.notes IS NOT excluded.notes
			    OR credit_contracts.expected_day IS NOT excluded.expected_day`,
			c.name, c.category, c.contractNumber, c.lender,
			c.originalPrincipalCents, c.startMonth, c.expectedEndMonth, c.notes, c.expectedDay,
		); err != nil {
			return fmt.Errorf("upsert credit contract %q: %w", c.name, err)
		}
		if keep[c.name] {
			return fmt.Errorf("credit %q is listed more than once", c.name)
		}
		keep[c.name] = true
	}

	return csvdata.DeleteUnlisted(log, q, "credit_contracts", "SELECT id, name FROM credit_contracts", keep,
		func(r *sql.Rows) (int64, string, error) {
			var id int64
			var name string
			if err := r.Scan(&id, &name); err != nil {
				return 0, "", err
			}
			return id, name, nil
		})
}

func creditPaymentKey(p creditPaymentSeed) string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s", p.month, p.paymentType, p.principalCents, p.interestCents, p.description, p.movement)
}

func mirrorCreditPayments(q database.Querier, rows [][]string) error {
	byContract := make(map[int64][]creditPaymentSeed)
	principalByContract := make(map[int64]money.Cents)
	originalByContract := make(map[int64]money.Cents)

	for i, row := range rows {
		if len(row) < 5 {
			return fmt.Errorf("row %d: expected at least 5 columns", i+2)
		}
		contractID, contract, err := getCreditContract(q, strings.TrimSpace(row[0]))
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		principal, err := csvdata.ParseNonNegativeCents(row[3], "principal_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		interest, err := csvdata.ParseNonNegativeCents(row[4], "interest_cents")
		if err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		payment := creditPaymentSeed{
			contractID:     contractID,
			month:          strings.TrimSpace(row[1]),
			paymentType:    strings.TrimSpace(row[2]),
			principalCents: principal,
			interestCents:  interest,
		}
		if len(row) >= 6 {
			payment.description = strings.TrimSpace(row[5])
		}
		if len(row) >= 7 {
			if payment.movement, err = csvdata.ParseMovementID(row[6]); err != nil {
				return fmt.Errorf("row %d: %w", i+2, err)
			}
		}
		if err := validateCreditPaymentSeed(contract, payment); err != nil {
			return fmt.Errorf("row %d: %w", i+2, err)
		}
		byContract[contractID] = append(byContract[contractID], payment)
		principalByContract[contractID] += principal
		originalByContract[contractID] = contract.originalPrincipalCents
	}
	for contractID, principal := range principalByContract {
		if principal > originalByContract[contractID] {
			return fmt.Errorf("credit contract %d: total principal payments exceed original principal", contractID)
		}
	}

	// Every contract is reconciled, so contracts with no rows end up with no payments.
	contractIDs, err := csvdata.QueryInt64s(q, "SELECT id FROM credit_contracts ORDER BY id")
	if err != nil {
		return fmt.Errorf("list credit contracts: %w", err)
	}
	for _, contractID := range contractIDs {
		desired := byContract[contractID]
		keys := make([]string, len(desired))
		for i, p := range desired {
			keys[i] = creditPaymentKey(p)
		}

		changed, err := csvdata.ClearIfChanged(q, "credit_payments", "contract_id", contractID,
			`SELECT month, type, principal_cents, interest_cents, COALESCE(description, ''), COALESCE(movement_sync_id, '')
			 FROM credit_payments WHERE contract_id = ?`,
			func(r *sql.Rows) (string, error) {
				var p creditPaymentSeed
				if err := r.Scan(&p.month, &p.paymentType, &p.principalCents, &p.interestCents, &p.description, &p.movement); err != nil {
					return "", err
				}
				return creditPaymentKey(p), nil
			}, keys)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		for _, p := range desired {
			if _, err := q.Exec(
				`INSERT INTO credit_payments
				 (contract_id, month, type, principal_cents, interest_cents, description, movement_sync_id)
				 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
				p.contractID, p.month, p.paymentType, p.principalCents, p.interestCents, p.description, p.movement,
			); err != nil {
				return fmt.Errorf("insert payment for credit contract %d month %s: %w", p.contractID, p.month, err)
			}
		}
	}
	return nil
}
