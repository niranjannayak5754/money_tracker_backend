run:
	@export $$(grep -v '^[#]' .env | xargs) && go run ./cmd/api
