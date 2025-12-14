package server

import (
	mongoPkg "go.mongodb.org/mongo-driver/mongo"

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
	Users      user.Service
	Categories category.Service
	Income     income.Service
	Expenses   expense.Service

	DB  *mongoPkg.Database
	Cfg config.Config

	AuthMw *middleware.AuthMiddleware
	AuthH  *handler.AuthHandler

	CategoryH *handler.CategoryHandler
	IncomeH   *handler.IncomeHandler
	ExpenseH  *handler.ExpenseHandler
	SummaryH  *handler.SummaryHandler
}

func BuildContainer(cfg config.Config, mc *mongo.Client) *Container {
	db := mc.Database(cfg.DBName)

	// repositories
	userRepo := user.NewMongoRepo(db)
	catRepo := category.NewMongoRepo(db)
	incRepo := income.NewMongoRepo(db)
	expRepo := expense.NewMongoRepo(db)
	summaryRepo := summaryrepo.New(db)

	// pure domain services
	userSvc := user.NewService(userRepo)
	categorySvc := category.NewService(catRepo)
	incomeSvc := income.NewService(incRepo)
	expenseSvc := expense.NewService(expRepo, catRepo)
	summarySvc := summary.NewService(summaryRepo)

	// middlewares
	authMw := middleware.NewAuthMiddleware(cfg.JWTSecret)

	// handlers
	authH := handler.NewAuthHandler(userSvc, cfg)
	summaryH := handler.NewSummaryHandler(summarySvc)
	categoryH := handler.NewCategoryHandler(categorySvc)
	incomeH := handler.NewIncomeHandler(incomeSvc)
	expenseH := handler.NewExpenseHandler(expenseSvc)

	return &Container{
		Users:      userSvc,
		Categories: categorySvc,
		Income:     incomeSvc,
		Expenses:   expenseSvc,
		DB:         db,
		Cfg:        cfg,

		AuthMw:    authMw,
		AuthH:     authH,
		SummaryH:  summaryH,
		CategoryH: categoryH,
		IncomeH:   incomeH,
		ExpenseH:  expenseH,
	}
}
