# money_tracker_backend
Backend Service for Money Tracking App in Go (Clean Architecture)

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


