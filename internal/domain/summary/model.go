package summary

type CategoryBreakdown struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
}

type Result struct {
	IncomeTotal       float64             `json:"income_total"`
	ExpenseTotal      float64             `json:"expense_total"`
	InvestmentTotal   float64             `json:"investment_total"`
	Savings           float64             `json:"savings"`
	NetWorth          float64             `json:"net_worth"`
	CategoryBreakdown []CategoryBreakdown `json:"category_breakdown"`
}

type MonthlyComparison struct {
	Month      string  `json:"month"`
	Income     float64 `json:"income"`
	Expense    float64 `json:"expense"`
	Investment float64 `json:"investment"`
	Savings    float64 `json:"savings"`
	NetWorth   float64 `json:"net_worth"`
}
