.PHONY: help up down build proto test-unit test-integration test benchmark index-sample index-corpus dev dev-docker dev-down

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
	@echo "Генерация gRPC кода для Gateway..."
	docker compose run --rm gateway python -m grpc_tools.protoc -I./protos \
		--python_out=./src/grpc_client --grpc_python_out=./src/grpc_client ./protos/search.proto ./protos/graph.proto
	@echo "Генерация gRPC кода для Search Engine Core..."
	docker compose run --rm search_engine_core python -m grpc_tools.protoc -I./protos \
		--python_out=./src --grpc_python_out=./src ./protos/search.proto ./protos/graph.proto
	@echo "Кодогенерация Protobuf успешно завершена!"

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
INDEXER_RUN = docker compose -f docker-compose.indexer.yml run --rm

index-sample:
	@mkdir -p data/raw data/index data/dump
	$(INDEXER_RUN) indexer sample

index-corpus:
	@mkdir -p data/index-benchmark data/dump
	$(INDEXER_RUN) -e INDEX_OUTPUT_PATH=/app/data/index-benchmark indexer corpus

# Бенчмарк (НИРС: Recall@k, MRR): сначала собирается индекс корпуса benchmark (секунды), затем метрики
benchmark: index-corpus
	docker compose run --rm search_engine_core python benchmark/evaluate.py
