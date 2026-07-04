package server

import (
	"log/slog"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/bankaccount"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/investment"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"

	// repositories (infra)
	bankaccountrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/bankaccount"
	categoryrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/category"
	expenserepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/expense"
	incomerepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/income"
	investmentrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/investment"
	sessionrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/session"
	summaryrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/summary"
	userrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/user"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/handler"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/middleware"
)

type Container struct {
	// domain services
	Users       user.Service
	Categories  category.Service
	Income      income.Service
	Expenses    expense.Service
	Summary     summary.Service
	Investment  investment.Service
	BankAccount bankaccount.Service

	// middleware
	AuthMw *middleware.AuthMiddleware

	// handlers
	AuthH        *handler.AuthHandler
	CategoryH    *handler.CategoryHandler
	IncomeH      *handler.IncomeHandler
	ExpenseH     *handler.ExpenseHandler
	SummaryH     *handler.SummaryHandler
	InvestmentH  *handler.InvestmentHandler
	BankAccountH *handler.BankAccountHandler
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

	userSvc := user.NewService(userRepo, sessionRepo)
	categorySvc := category.NewService(categoryRepo, expenseRepo)
	incomeSvc := income.NewService(incomeRepo)
	expenseSvc := expense.NewService(expenseRepo, categoryRepo)
	summarySvc := summary.NewService(summaryRepo)
	investmentSvc := investment.NewService(investmentRepo)
	bankAccountSvc := bankaccount.NewService(bankAccountRepo)

	authMw := middleware.NewAuthMiddleware(cfg.JWTSecret)

	authH := handler.NewAuthHandler(userSvc, cfg, logger)
	categoryH := handler.NewCategoryHandler(categorySvc, logger)
	incomeH := handler.NewIncomeHandler(incomeSvc, logger)
	expenseH := handler.NewExpenseHandler(expenseSvc, logger)
	summaryH := handler.NewSummaryHandler(summarySvc, logger)
	investmentH := handler.NewInvestmentHandler(investmentSvc, logger)
	bankAccountH := handler.NewBankAccountHandler(bankAccountSvc, logger)

	return &Container{
		Users:       userSvc,
		Categories:  categorySvc,
		Income:      incomeSvc,
		Expenses:    expenseSvc,
		Summary:     summarySvc,
		Investment:  investmentSvc,
		BankAccount: bankAccountSvc,

		AuthMw: authMw,

		AuthH:        authH,
		CategoryH:    categoryH,
		IncomeH:      incomeH,
		ExpenseH:     expenseH,
		SummaryH:     summaryH,
		InvestmentH:  investmentH,
		BankAccountH: bankAccountH,
	}
}
