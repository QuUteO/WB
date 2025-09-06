docker-compose up -d



migrate -path ./db/migrations -database "postgres://root:1234@localhost:5432/postgres?sslmode=disable" up




go run cmd/main.go --config=./config/config.yaml
