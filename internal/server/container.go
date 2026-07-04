package server

import (
	"log/slog"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/bankaccount"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/budget"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/debt"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/goal"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"

	// repositories (infra)
	bankaccountrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/bankaccount"
	budgetrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/budget"
	categoryrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/category"
	debtrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/debt"
	expenserepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/expense"
	goalrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/goal"
	incomerepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/income"
	investmentrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/investment"
	notificationrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/notification"
	recurringrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/recurring"
	sessionrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/session"
	summaryrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/summary"
	userrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/user"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/handler"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/middleware"
)

type Container struct {
	// domain services
	Users        user.Service
	Categories   category.Service
	Income       income.Service
	Expenses     expense.Service
	Summary      summary.Service
	Investment   investment.Service
	BankAccount  bankaccount.Service
	Debt         debt.Service
	Recurring    recurring.Service
	Budget       budget.Service
	Goal         goal.Service
	Notification notification.Service

	// middleware
	AuthMw *middleware.AuthMiddleware

	// handlers
	AuthH         *handler.AuthHandler
	CategoryH     *handler.CategoryHandler
	IncomeH       *handler.IncomeHandler
	ExpenseH      *handler.ExpenseHandler
	SummaryH      *handler.SummaryHandler
	InvestmentH   *handler.InvestmentHandler
	BankAccountH  *handler.BankAccountHandler
	DebtH         *handler.DebtHandler
	RecurringH    *handler.RecurringHandler
	BudgetH       *handler.BudgetHandler
	GoalH         *handler.GoalHandler
	NotificationH *handler.NotificationHandler
}

func BuildContainer(
	cfg config.Config,
	mc *mongo.Client,
	logger *slog.Logger,
) *Container {

	db := mc.Database(cfg.DBName)

	userRepo := userrepo.New(db, logger)
	categoryRepo := categoryrepo.New(db, logger)
	incomeRepo := incomerepo.New(db, logger)
	expenseRepo := expenserepo.New(db, logger)
	summaryRepo := summaryrepo.New(db, logger)
	investmentRepo := investmentrepo.New(db, logger)
	sessionRepo := sessionrepo.New(db, logger)
	bankAccountRepo := bankaccountrepo.New(db, logger)
	debtRepo := debtrepo.New(db, logger)
	recurringRepo := recurringrepo.New(db, logger)
	budgetRepo := budgetrepo.New(db, logger)
	goalRepo := goalrepo.New(db, logger)
	notificationRepo := notificationrepo.New(db, logger)

	userSvc := user.NewService(userRepo, sessionRepo)
	categorySvc := category.NewService(categoryRepo, expenseRepo, incomeRepo)
	incomeSvc := income.NewService(incomeRepo, categoryRepo)
	budgetSvc := budget.NewService(budgetRepo, categoryRepo)
	notificationSvc := notification.NewService(notificationRepo)
	expenseSvc := expense.NewService(expenseRepo, categoryRepo, budgetSvc, notificationSvc)
	summarySvc := summary.NewService(summaryRepo, budgetSvc)
	investmentSvc := investment.NewService(investmentRepo)
	bankAccountSvc := bankaccount.NewService(bankAccountRepo)
	debtSvc := debt.NewService(debtRepo)
	recurringSvc := recurring.NewService(recurringRepo)
	goalSvc := goal.NewService(goalRepo, bankAccountRepo, investmentRepo)

	authMw := middleware.NewAuthMiddleware(cfg.JWTSecret)

	authH := handler.NewAuthHandler(userSvc, cfg, logger)
	categoryH := handler.NewCategoryHandler(categorySvc, logger)
	incomeH := handler.NewIncomeHandler(incomeSvc, logger)
	expenseH := handler.NewExpenseHandler(expenseSvc, logger)
	summaryH := handler.NewSummaryHandler(summarySvc, logger)
	investmentH := handler.NewInvestmentHandler(investmentSvc, logger)
	bankAccountH := handler.NewBankAccountHandler(bankAccountSvc, logger)
	debtH := handler.NewDebtHandler(debtSvc, logger)
	recurringH := handler.NewRecurringHandler(recurringSvc, logger)
	budgetH := handler.NewBudgetHandler(budgetSvc, logger)
	goalH := handler.NewGoalHandler(goalSvc, logger)
	notificationH := handler.NewNotificationHandler(notificationSvc, logger)

	return &Container{
		Users:        userSvc,
		Categories:   categorySvc,
		Income:       incomeSvc,
		Expenses:     expenseSvc,
		Summary:      summarySvc,
		Investment:   investmentSvc,
		BankAccount:  bankAccountSvc,
		Debt:         debtSvc,
		Recurring:    recurringSvc,
		Budget:       budgetSvc,
		Goal:         goalSvc,
		Notification: notificationSvc,

		AuthMw: authMw,

		AuthH:         authH,
		CategoryH:     categoryH,
		IncomeH:       incomeH,
		ExpenseH:      expenseH,
		SummaryH:      summaryH,
		InvestmentH:   investmentH,
		BankAccountH:  bankAccountH,
		DebtH:         debtH,
		RecurringH:    recurringH,
		BudgetH:       budgetH,
		GoalH:         goalH,
		NotificationH: notificationH,
	}
}
