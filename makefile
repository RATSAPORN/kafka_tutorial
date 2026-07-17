.PHONY: clean tidy run all migrate-up migrate-down compose-up compose-down setup-db deploy

clean:
	go clean -modcache
	kubectl delete namespace myapps


tidy:
	go mod tidy

run:
	go run ./cmd/main.go

# Migrate up (apply all .up.sql)
run-up:
	go run ./cmd/main.go up

# Rollback last N migrations (default 1 if not specified)
# Usage: make run-down steps=2
run-down:
	go run ./cmd/main.go down $(steps)


all: clean tidy run

# Migration commands
migrate-up: 
	@echo "Running database migration UP..."
	@chmod +x scripts/run_migration.sh
	@./scripts/run_migration.sh up

migrate-down: 
	@echo "Running database migration DOWN..."
	@chmod +x scripts/run_migration.sh
	@./scripts/run_migration.sh down

# Setup database and run migrations
setup-db: migrate-up
	@echo "Database setup completed!"

# Reset database (drop and recreate tables)
reset-db: migrate-down migrate-up
	@echo "Database reset completed!"

compose-up:
	@docker compose up -d
	@echo "Database setup completed!"

compose-down:
	@docker-compose down -v
	@echo "Database reset completed!"

deploy:
	kubectl apply -f namespace.yaml
	kubectl apply -f configmap.yaml
	kubectl apply -f secrets.yaml

	kubectl apply -f zookeeper.yaml
	kubectl rollout status deployment/zookeeper -n myapps

	kubectl apply -f kafka.yaml
	kubectl rollout status deployment/kafka -n myapps

	kubectl apply -f postgres-app1.yaml
	kubectl rollout status deployment/postgres-app1 -n myapps

	kubectl apply -f deployment.yaml
	kubectl rollout status deployment/app1 -n myapps

	kubectl apply -f kong-db.yaml
	kubectl rollout status deployment/kong-db -n myapps

	kubectl apply -f kong-migration.yaml
	kubectl wait --for=condition=complete job/kong-migrations-bootstrap -n myapps --timeout=300s

	kubectl apply -f kong.yaml
	kubectl rollout status deployment/kong -n myapps

	kubectl apply -f kong-deck.yaml
	kubectl wait --for=condition=complete job/kong-config-sync -n myapps --timeout=300s