# money_tracker_backend
Backend Service for Money Tracking App in Go (Clean Architecture)

> This is the `release-expense-tracker` branch — a focused expense tracker,
> not the full money tracker on `release`/`main` (no income, investment,
> debt, or net-worth tracking here).

## Features

- **Expenses** — create/edit/delete, tagged to a category (merchant, notes,
  tags), filterable by month
- **Categories** — expense-only, with color/icon, archive/unarchive, and
  reassign-and-archive (move all expenses off a category before archiving)
- **Budgets** — per-category monthly caps plus an overall monthly cap,
  effective-dated, with spent/budgeted/percent-used and an exceeded flag
- **Savings accounts** — name, balance, optional interest rate, with atomic
  add/withdraw actions (audit-logged, guarded against overdrawing) and a
  full ledger of every balance change; no direct balance editing
- **Goals** — linked to a savings account; progress is derived live from
  the account's current balance, so an add/withdraw immediately moves it
- **Recurring expenses** — monthly/yearly templates that auto-materialize
  into real expense records on schedule; pause/resume without touching past
  occurrences
- **Notifications** — budget-exceeded and upcoming-bill-due alerts
- **Summary/analytics** — monthly expense total, current savings balance,
  category breakdown, and a monthly expense trend across the last N months
- Dates are stored and returned as plain calendar values (no timezone
  conversion), so the date you pick is the date that's saved

To run this service, Clone this repository & Follow these steps:

1. First Change Directory to money_tracker_backend

2. Using Docker:
     - docker compose up -d             (start dev environment -> this includes hot reload capability)
     - docker compose ps                (show service status)
     - docker compose logs -f api       (for api logs)
     - docker compose down              (stop everything -> removes container + networks)
     - docker compose down -v           (removes container + networks + clear volumes -> delete DB data)
     - docker system prune -a --volumes (full cleanup -> removes unused containers & unused volumes)

3. Run Using go commands:
     - Open Terminal (shell)
     - Run `cp .env.example .env`
     - Open .env file and change `MONGO_URI=mongodb://mongo:27017` to `MONGO_URI=mongodb://localhost:27017` and Save .env File
     - Run Command: `make run` in terminal
     - HEALTH CHECK ROUTE: `curl http://localhost:8080/health`
     - Press `CTRL + c` to exit

4. When server is up and running:
     Check for this route in server: `http://localhost:8080/docs` for swagger ui.

# Code Architecture
cmd/
 └── api/
     └── main.go

internal/
 ├── config/
 |
 ├── logging/
 |
 ├── apperr/                       // app errors
 |
 ├── domain/
 │    └── <domain>/
 │         ├── model.go
 │         ├── repository.go      // interface
 │         └── service.go
 |
 ├── repository/
 |    ├── errors.go
 │    └── <domain>/
 │         └── mongo.go           // mongo implementation
 |
 ├── http/
 │    ├── handler/
 │    ├── middleware/
 │    ├── requestctx/
 │    └── response/
 |        ├── respond.go
 |        └── error.go
 |
 ├── platform/
 │    └── mongo/
 │         └── client.go
 │
 ├── security/
 │    └── jwt.go
 |
 └── server/
      ├── router.go
      ├── server.go
      └── container.go
      

# Dependency Direction
main → server
server → http
http → security
http → domain → repository → platform


