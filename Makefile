.PHONY: help up down build proto test-unit test-integration test benchmark

# По умолчанию выводим справку
help:
	@echo "==================== Errorium Makefile ===================="
	@echo "Доступные команды:"
	@echo "  make up                - Собрать и запустить все сервисы системы"
	@echo "  make down              - Остановить и удалить контейнеры"
	@echo "  make build             - Пересобрать Docker-образы сервисов"
	@echo "  make proto             - Скомпилировать Protobuf контракты (gRPC) через Docker"
	@echo "  make test-unit         - Запустить модульные тесты через Docker"
	@echo "  make test-integration  - Запустить интеграционные тесты через Docker"
	@echo "  make test              - Запустить все тесты (unit + integration)"
	@echo "  make benchmark         - Запустить расчёт метрик Recall@k и MRR (НИРС)"
	@echo "==========================================================="

# Запуск и управление контейнерами
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
		--python_out=./src/grpc_client --grpc_python_out=./src/grpc_client ./protos/search.proto
	@echo "Генерация gRPC кода для Search Engine..."
	docker compose run --rm search_engine python -m grpc_tools.protoc -I./protos \
		--python_out=./src --grpc_python_out=./src ./protos/search.proto
	@echo "Кодогенерация Protobuf успешно завершена!"

# Тестирование
test-unit:
	docker compose run --rm gateway pytest tests/unit

test-integration:
	docker compose run --rm gateway pytest tests/integration

test: test-unit test-integration

# Бенчмарк (НИРС: Recall@k, MRR)
benchmark:
	docker compose run --rm search_engine python benchmark/evaluate.py
