package server

import (
	mongoPkg "go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/config"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"
	"github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
)

type Container struct {
	Users      user.Service
	Categories category.Service
	Income     income.Service
	Expenses   expense.Service
	DB         *mongoPkg.Database
	Cfg        config.Config
}

func BuildContainer(cfg config.Config, mc *mongo.Client) *Container {
	db := mc.Database(cfg.DBName)

	// repositories
	userRepo := user.NewMongoRepo(db)
	catRepo := category.NewMongoRepo(db)
	incRepo := income.NewMongoRepo(db)
	expRepo := expense.NewMongoRepo(db)

	// pure domain services
	userSvc := user.NewService(userRepo)
	categorySvc := category.NewService(catRepo)
	incomeSvc := income.NewService(incRepo)
	expenseSvc := expense.NewService(expRepo, catRepo)

	return &Container{
		Users:      userSvc,
		Categories: categorySvc,
		Income:     incomeSvc,
		Expenses:   expenseSvc,
		DB:         db,
		Cfg:        cfg,
	}
}
