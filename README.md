# money_tracker_backend
Backend Service for Money Tracking App in Go (Clean Architecture)

# Code Architecture
cmd/
 └── api/
     └── main.go

internal/
 ├── config/

 ├── domain/
 │    └── <domain>/
 │         ├── model.go
 │         ├── repository.go      // interface
 │         └── service.go

 ├── repository/
 │    └── <domain>/
 │         └── mongo.go           // mongo implementation

 ├── http/
 │    ├── handler/
 │    ├── middleware/
 │    ├── context/
 │    └── respond.go

 ├── platform/
 │    ├── mongo/
 │
 │
 ├── security/
 │    └── jwt.go

 └── server/
      ├── router.go
      ├── server.go
      ├── container.go
      

# Dependency Direction
main → server
server → http
http → security
http → domain → repository → platform


