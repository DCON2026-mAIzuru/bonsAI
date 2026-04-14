SHELL := /bin/bash

.PHONY: dev-up dev-down dev-status docker-up docker-down docker-logs docker-mac-up docker-mac-down docker-mac-logs docker-mac-status

dev-up:
	./scripts/dev-up.sh

dev-down:
	./scripts/dev-down.sh

dev-status:
	./scripts/dev-status.sh

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-mac-up:
	docker compose -f docker-compose.mac.yml up --build -d

docker-mac-down:
	docker compose -f docker-compose.mac.yml down

docker-mac-logs:
	docker compose -f docker-compose.mac.yml logs -f

docker-mac-status:
	docker compose -f docker-compose.mac.yml ps
