.PHONY: help up down build proto test-unit test-integration test benchmark dev dev-docker dev-down

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
	@echo "Интеграционные тесты пока не сконфигурированы"

test: test-unit

# Бенчмарк (НИРС: Recall@k, MRR)
benchmark:
	docker compose run --rm search_engine_core python benchmark/evaluate.py
