package server

import (
	"log/slog"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/bankaccount"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/budget"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/goal"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"

	// repositories (infra)
	bankaccountrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/bankaccount"
	budgetrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/budget"
	categoryrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/category"
	expenserepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/expense"
	goalrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/goal"
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
	Expenses     expense.Service
	Summary      summary.Service
	BankAccount  bankaccount.Service
	Recurring    recurring.Service
	Budget       budget.Service
	Goal         goal.Service
	Notification notification.Service

	// middleware
	AuthMw *middleware.AuthMiddleware

	// handlers
	AuthH         *handler.AuthHandler
	CategoryH     *handler.CategoryHandler
	ExpenseH      *handler.ExpenseHandler
	SummaryH      *handler.SummaryHandler
	BankAccountH  *handler.BankAccountHandler
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
	expenseRepo := expenserepo.New(db, logger)
	summaryRepo := summaryrepo.New(db, logger)
	sessionRepo := sessionrepo.New(db, logger)
	bankAccountRepo := bankaccountrepo.New(db, logger)
	recurringRepo := recurringrepo.New(db, logger)
	budgetRepo := budgetrepo.New(db, logger)
	goalRepo := goalrepo.New(db, logger)
	notificationRepo := notificationrepo.New(db, logger)

	userSvc := user.NewService(userRepo, sessionRepo)
	categorySvc := category.NewService(categoryRepo, expenseRepo)
	budgetSvc := budget.NewService(budgetRepo, categoryRepo)
	notificationSvc := notification.NewService(notificationRepo)
	expenseSvc := expense.NewService(expenseRepo, categoryRepo, budgetSvc, notificationSvc)
	summarySvc := summary.NewService(summaryRepo, budgetSvc)
	bankAccountSvc := bankaccount.NewService(bankAccountRepo)
	recurringSvc := recurring.NewService(recurringRepo)
	goalSvc := goal.NewService(goalRepo, bankAccountRepo)

	authMw := middleware.NewAuthMiddleware(cfg.JWTSecret)

	authH := handler.NewAuthHandler(userSvc, cfg, logger)
	categoryH := handler.NewCategoryHandler(categorySvc, logger)
	expenseH := handler.NewExpenseHandler(expenseSvc, logger)
	summaryH := handler.NewSummaryHandler(summarySvc, logger)
	bankAccountH := handler.NewBankAccountHandler(bankAccountSvc, logger)
	recurringH := handler.NewRecurringHandler(recurringSvc, logger)
	budgetH := handler.NewBudgetHandler(budgetSvc, logger)
	goalH := handler.NewGoalHandler(goalSvc, logger)
	notificationH := handler.NewNotificationHandler(notificationSvc, logger)

	return &Container{
		Users:        userSvc,
		Categories:   categorySvc,
		Expenses:     expenseSvc,
		Summary:      summarySvc,
		BankAccount:  bankAccountSvc,
		Recurring:    recurringSvc,
		Budget:       budgetSvc,
		Goal:         goalSvc,
		Notification: notificationSvc,

		AuthMw: authMw,

		AuthH:         authH,
		CategoryH:     categoryH,
		ExpenseH:      expenseH,
		SummaryH:      summaryH,
		BankAccountH:  bankAccountH,
		RecurringH:    recurringH,
		BudgetH:       budgetH,
		GoalH:         goalH,
		NotificationH: notificationH,
	}
}
