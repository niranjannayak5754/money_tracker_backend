package summary

type CategoryBreakdown struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
}

// BudgetStatus reports actual spend against a budgeted amount. CategoryID
// is empty for the overall cap. A category with no budget set simply
// doesn't appear here, rather than showing a misleading 0%/not-exceeded.
type BudgetStatus struct {
	CategoryID  string  `json:"category_id,omitempty"`
	Budgeted    float64 `json:"budgeted"`
	Spent       float64 `json:"spent"`
	PercentUsed float64 `json:"percent_used"`
	Exceeded    bool    `json:"exceeded"`
}

type Result struct {
	IncomeTotal       float64             `json:"income_total"`
	ExpenseTotal      float64             `json:"expense_total"`
	InvestmentTotal   float64             `json:"investment_total"`
	Savings           float64             `json:"savings"`
	NetWorth          float64             `json:"net_worth"`
	CategoryBreakdown []CategoryBreakdown `json:"category_breakdown"`
	BudgetStatus      []BudgetStatus      `json:"budget_status,omitempty"`
	OverallBudget     *BudgetStatus       `json:"overall_budget,omitempty"`
}

type MonthlyComparison struct {
	Month         string         `json:"month"`
	Income        float64        `json:"income"`
	Expense       float64        `json:"expense"`
	Investment    float64        `json:"investment"`
	Savings       float64        `json:"savings"`
	NetWorth      float64        `json:"net_worth"`
	BudgetStatus  []BudgetStatus `json:"budget_status,omitempty"`
	OverallBudget *BudgetStatus  `json:"overall_budget,omitempty"`
}
