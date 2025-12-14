package server

import (
	"log/slog"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"

	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	summaryrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/summary"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/handler"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/middleware"
)

type Container struct {
	// domain services
	Users      user.Service
	Categories category.Service
	Income     income.Service
	Expenses   expense.Service

	// http layer
	AuthMw    *middleware.AuthMiddleware
	AuthH     *handler.AuthHandler
	CategoryH *handler.CategoryHandler
	IncomeH   *handler.IncomeHandler
	ExpenseH  *handler.ExpenseHandler
	SummaryH  *handler.SummaryHandler
}

func BuildContainer(
	cfg config.Config,
	mc *mongo.Client,
	logger *slog.Logger,
) *Container {

	db := mc.Database(cfg.DBName)

	// repositories
	userRepo := user.NewMongoRepo(db)
	catRepo := category.NewMongoRepo(db)
	incRepo := income.NewMongoRepo(db)
	expRepo := expense.NewMongoRepo(db)
	summaryRepo := summaryrepo.New(db)

	// domain services
	userSvc := user.NewService(userRepo)
	categorySvc := category.NewService(catRepo)
	incomeSvc := income.NewService(incRepo)
	expenseSvc := expense.NewService(expRepo, catRepo)
	summarySvc := summary.NewService(summaryRepo)

	// middlewares
	authMw := middleware.NewAuthMiddleware(cfg.JWTSecret)

	// handlers
	authH := handler.NewAuthHandler(userSvc, cfg, logger)
	categoryH := handler.NewCategoryHandler(categorySvc, logger)
	incomeH := handler.NewIncomeHandler(incomeSvc, logger)
	expenseH := handler.NewExpenseHandler(expenseSvc, logger)
	summaryH := handler.NewSummaryHandler(summarySvc, logger)

	return &Container{
		Users:      userSvc,
		Categories: categorySvc,
		Income:     incomeSvc,
		Expenses:   expenseSvc,

		AuthMw:    authMw,
		AuthH:     authH,
		CategoryH: categoryH,
		IncomeH:   incomeH,
		ExpenseH:  expenseH,
		SummaryH:  summaryH,
	}
}
