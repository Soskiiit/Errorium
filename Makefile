-include .env

export PATH := $(HOME)/go/bin:$(CURDIR)/.venv/bin:$(PATH)

.PHONY: help up down build proto test-unit test-integration test benchmark index-sample index-corpus dev dev-docker dev-down ghcr-build-push ghcr-deploy

GHCR_OWNER ?= soskiiit
IMAGE_TAG ?= develop
SERVER_USER ?= root
SERVER_HOST ?=
SERVER_DIR ?= ~/Errorium

# По умолчанию выводим справку
help:
	@echo "==================== Errorium Makefile ===================="
	@echo "Доступные команды:"
	@echo "  make dev               - Локальный запуск дев-сервера SPA (:3000, без Docker)"
	@echo "  make dev-docker        - Локальный запуск стека в Docker (:3000 фронтенд + :8000 gateway)"
	@echo "  make dev-down          - Остановка локального дев-стека Docker"
	@echo "  make up                - Собрать и запустить продакшн сервисы (SSL :80/:443)"
	@echo "  make down              - Остановить и удалить контейнеры"
	@echo "  make build             - Пересобрать Docker-образы сервисов"
	@echo "  make proto             - Скомпилировать Protobuf контракты (gRPC) через Docker"
	@echo "  make test-unit         - Запустить модульные тесты"
	@echo "  make test-integration  - Запустить интеграционные тесты"
	@echo "  make test              - Запустить все тесты (unit + integration)"
	@echo "  make benchmark         - Запустить расчёт метрик Recall@k и MRR (НИРС)"
	@echo "  make index-sample      - Индексатор: выборка из дампа SO (SO_DUMP_DIR, см. .env.example)"
	@echo "  make index-corpus      - Индексатор: индекс малого корпуса benchmark (секунды)"
	@echo "  make ghcr-build-push   - Собрать образы на хосте и отправить в GHCR (ghcr.io)"
	@echo "  make ghcr-deploy       - Полный цикл: сборка -> push в GHCR -> деплой на сервер"
	@echo "==========================================================="

# Локальная разработка (Dev-сервер)
dev:
	@echo "Запуск локального дев-сервера Errorium SPA..."
	python3 services/frontend/dev_server.py

dev-docker:
	docker compose -f docker-compose.dev.yml up --build

dev-down:
	docker compose -f docker-compose.dev.yml down

# Запуск и управление продакшн контейнерами
up:
	docker compose up --build

down:
	docker compose down


build:
	docker compose build

# Кодогенерация gRPC контрактов
proto:
	protoc -I protos \
	  --go_out=services/gateway/internal/pb --go_opt=module=github.com/soskiiit/errorium/protos \
	  --go-grpc_out=services/gateway/internal/pb --go-grpc_opt=module=github.com/soskiiit/errorium/protos \
	  protos/search.proto protos/graph.proto
	python3 -m grpc_tools.protoc -I protos \
	  --python_out=services/search_engine_core/src \
	  --grpc_python_out=services/search_engine_core/src \
	  protos/search.proto protos/graph.proto
	@python3 -c "import glob, re; [None for p in glob.glob('services/search_engine_core/src/*_pb2.py') for s in [open(p).read()] if open(p, 'w').write(re.sub(r'6,\s*31,\s*1,', '5,\n    29,\n    5,', re.sub(r'6\.31\.1', '5.29.5', s)))]"

# Тестирование
test-unit:
	@echo "Запуск unit-тестов Gateway (Go)..."
	cd services/gateway && go test -v -race ./...
	@echo "Запуск unit-тестов Frontend SPA & архитектуры (Python)..."
	python3 -m unittest discover -s tests/unit -p "test_*.py"

test-integration:
	@echo "Запуск интеграционных тестов (Python)..."
	python3 -m unittest discover -s tests/integration -p "test_*.py"

test: test-unit

# Индексатор (Роль 1): офлайн-задачи в контейнере, отдельный compose-файл
INDEXER_RUN = docker compose -f docker-compose.indexer.yml run --build --rm

index-sample:
	@mkdir -p data/raw data/index data/dump
	$(INDEXER_RUN) indexer sample

index-corpus:
	@mkdir -p data/index-benchmark data/dump
	$(INDEXER_RUN) -e INDEX_OUTPUT_PATH=/app/data/index-benchmark indexer corpus

# Бенчмарк (НИРС: Recall@k, MRR): сначала собирается индекс корпуса benchmark (секунды), затем метрики
benchmark: index-corpus
	docker compose run --rm -e INDEX_DIR=/app/data/index-benchmark search_engine_core python benchmark/evaluate.py

# Сборка образов на хосте под linux/amd64 и отправка в GitHub Container Registry (GHCR)
ghcr-build-push:
	@echo "==> Сборка и отправка образов в GHCR (owner: $(GHCR_OWNER), tag: $(IMAGE_TAG))..."
	docker buildx build --platform linux/amd64 -t ghcr.io/$(GHCR_OWNER)/errorium-gateway:$(IMAGE_TAG) ./services/gateway --push
	docker buildx build --platform linux/amd64 -t ghcr.io/$(GHCR_OWNER)/errorium-frontend:$(IMAGE_TAG) ./services/frontend --push
	@echo "==> Образы успешно собраны и отправлены в GHCR!"

# Полный цикл: локальная сборка -> отправка в GHCR -> обновление и запуск на сервере по SSH
ghcr-deploy: ghcr-build-push
	@if [ -z "$(SERVER_HOST)" ]; then \
		echo "Ошибка: Не указан адрес сервера (SERVER_HOST)."; \
		echo "Задайте в .env или передайте аргументом: make ghcr-deploy SERVER_HOST=x.x.x.x [SERVER_USER=root]"; \
		exit 1; \
	fi
	@echo "==> Синхронизация конфигурации и перезапуск сервисов на $(SERVER_HOST)..."
	ssh $(SERVER_USER)@$(SERVER_HOST) "mkdir -p $(SERVER_DIR) && if [ -d $(SERVER_DIR)/deploy/garage/garage.toml ]; then rm -rf $(SERVER_DIR)/deploy/garage/garage.toml; fi"
	scp docker-compose.yml $(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/docker-compose.yml
	scp -r deploy $(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/
	ssh $(SERVER_USER)@$(SERVER_HOST) "cd $(SERVER_DIR) && docker compose pull && docker compose up -d --remove-orphans"
	@echo "==> Деплой на $(SERVER_HOST) успешно завершён!"
