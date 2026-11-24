package cashflow

// Roles an expense category can have. A role says where the money goes, so Home
// can warn when what the category spent is not matched by what was typed in the
// module of that role. A category has at most one role.
const (
	// RoleSavings marks money put into savings.
	RoleSavings = "savings"
	// RoleInvestments marks money put into investments.
	RoleInvestments = "investments"
	// RoleGoals marks money put into goals.
	RoleGoals = "goals"
	// RoleCredits marks money paid on credits.
	RoleCredits = "credits"
)

// PutAsideRoles are the roles whose money counts as put aside: kept for the
// household's future. Credit payments do not count, because the asset they pay
// for (a house, say) is not recorded anywhere.
var PutAsideRoles = []string{RoleSavings, RoleInvestments, RoleGoals}
