##@ Development

up: ## Prepare local development
	docker-compose up -d

build: ## Build shibc-faucet binary
	go build -o shibc-faucet

db: ## Access to local db
	docker exec -it shibc-faucet-db /bin/bash -c 'psql -U shibc -d shibc_faucet'