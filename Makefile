BACKEND_DIR=backend
FRONTEND_DIR=frontend

.PHONY: backend frontend test

backend:
	cd $(BACKEND_DIR) && go run ./cmd/api

frontend:
	cd $(FRONTEND_DIR) && npm run dev

test:
	cd $(BACKEND_DIR) && go test ./...
