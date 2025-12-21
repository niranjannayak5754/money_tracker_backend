package summary

type CategoryBreakdown struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
}

type Result struct {
	IncomeTotal       float64             `json:"income_total"`
	ExpenseTotal      float64             `json:"expense_total"`
	Savings           float64             `json:"savings"`
	CategoryBreakdown []CategoryBreakdown `json:"category_breakdown"`
}

type MonthlyComparison struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Savings float64 `json:"savings"`
}
