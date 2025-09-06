docker-compose up -d



migrate -path ./db/migrations -database "postgres://root:1234@localhost:5432/postgres?sslmode=disable" up




go run cmd/main.go --config=./config/config.yaml








https://github.com/user-attachments/assets/0fdae541-a5eb-4b16-93b3-3a8955932edd


