BACKEND_DIR=backend
FRONTEND_DIR=frontend

.PHONY: backend frontend test compose-up compose-down compose-logs

backend:
	cd $(BACKEND_DIR) && go run ./cmd/api

frontend:
	cd $(FRONTEND_DIR) && npm run dev

test:
	cd $(BACKEND_DIR) && go test ./...

compose-up:
	docker compose up --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f
