.PHONY: build-cli
build-cli:
	go build -o ./bin/goflow ./tools/cli/cmd/goflow-cli/main.go

.PHONY: install-cli
install-cli:
	go install ./tools/cli/cmd/goflow-cli

REPORTS_DIR := reports
ALLURE_RESULTS := $(REPORTS_DIR)/allure-results
ALLURE_REPORT := $(REPORTS_DIR)/allure-report

.PHONY: test
test:
	@echo "=== Запуск Лондонских тестов (TransactionService) ==="
	go test -v ./services/transactions/internal/service/...
	@echo ""
	@echo "=== Запуск Классических тестов (AccountService + Real Postgres Repos + pgxmock) ==="
	go test -v -run TestAccountService_Classic ./services/accounts/internal/services/...

.PHONY: test-random
test-random:
	@echo "=== Запуск тестов в случайном порядке (-shuffle=on) ==="
	go test -v -shuffle=on ./services/transactions/internal/service/...
	go test -v -shuffle=on -run TestAccountService_Classic ./services/accounts/internal/services/...

.PHONY: test-offline
test-offline:
	@echo "=== Запуск тестов в автономном режиме (без внешних зависимостей) ==="
	GOPROXY=off go test -v ./services/transactions/internal/service/...
	GOPROXY=off go test -v -run TestAccountService_Classic ./services/accounts/internal/services/...

export PATH := $(shell go env GOPATH)/bin:$(PATH)

COVERPKG_TX := transactions/...
COVERPKG_ACC := accounts/...,shared/...

.PHONY: test-coverage
test-coverage:
	@mkdir -p $(REPORTS_DIR)
	@go test -coverprofile=$(REPORTS_DIR)/coverage-tx.out -covermode=atomic -coverpkg=$(COVERPKG_TX) ./services/transactions/internal/service/... > /dev/null
	@go test -coverprofile=$(REPORTS_DIR)/coverage-acc.out -covermode=atomic -coverpkg=$(COVERPKG_ACC) -run TestAccountService_Classic ./services/accounts/internal/services/... > /dev/null
	@echo "mode: atomic" > $(REPORTS_DIR)/coverage.out
	@tail -n +2 $(REPORTS_DIR)/coverage-tx.out >> $(REPORTS_DIR)/coverage.out
	@tail -n +2 $(REPORTS_DIR)/coverage-acc.out >> $(REPORTS_DIR)/coverage.out
	@grep -v "/mocks/" $(REPORTS_DIR)/coverage.out | grep -v "shared/pkg/db/postgres/pool.go" > $(REPORTS_DIR)/coverage-clean.out
	@echo "=== Сводка покрытия по функциям и блокам (ветвлениям) ==="
	@go tool cover -func=$(REPORTS_DIR)/coverage-clean.out

.PHONY: test-coverage-html
test-coverage-html: test-coverage
	@go tool cover -html=$(REPORTS_DIR)/coverage-clean.out -o $(REPORTS_DIR)/coverage-std.html
	@gocov convert $(REPORTS_DIR)/coverage-clean.out | gocov-html -t kit > $(REPORTS_DIR)/coverage.html
	@echo "Информативный HTML-отчет (gocov-html) сохранен в: $(REPORTS_DIR)/coverage.html"
	@echo "Стандартный HTML-отчет (go tool cover) сохранен в: $(REPORTS_DIR)/coverage-std.html"

.PHONY: test-coverage-open
test-coverage-open: test-coverage-html
	@xdg-open $(REPORTS_DIR)/coverage.html 2>/dev/null || open $(REPORTS_DIR)/coverage.html 2>/dev/null || echo "Откройте в браузере: file://$(abspath $(REPORTS_DIR)/coverage.html)"

.PHONY: test-report
test-report:
	@echo "=== Генерация Allure-отчета с пирамидой тестирования и проверкой Requirement 8 ==="
	@./scripts/generate_allure_report.sh


.PHONY: test-report-open
test-report-open:
	npx -y allure-commandline open $(ALLURE_REPORT)

.PHONY: clean-reports
clean-reports:
	rm -rf $(REPORTS_DIR)

.PHONY: test-process-trace
test-process-trace:
	@echo "=== Трассировка создания процессов ОС (execve) при запуске тестов ==="
	@strace -f -q -e trace=execve go test -count=1 ./services/transactions/internal/service ./services/accounts/internal/services 2>&1 | grep --color=never '\.test"' || true

.PHONY: lint-api
lint-api:
	npx -y @redocly/cli lint docs/openapi/openapi.yaml

.PHONY: mock-api
mock-api:
	npx -y -p @faker-js/faker -p @stoplight/prism-cli prism mock docs/openapi/openapi.yaml -p 4010

.PHONY: test-api-mock
test-api-mock:
	./scripts/test_mock.sh

# Тестовый стенд и окружение ЛР №2 (Docker, Test Stand, Teardown)
.PHONY: test-env-up
test-env-up:
	@echo "=== Запуск тестовой инфраструктуры (PostgreSQL + Redis) ==="
	docker compose -f docker-compose.test.yml up -d --wait postgres-test redis-test
	@echo "=== Применение миграций и инициализация тестовых БД ==="
	./scripts/test_init.sh

.PHONY: test-env-down
test-env-down:
	@echo "=== Остановка и удаление тестового стенда (с томами) ==="
	docker compose -f docker-compose.test.yml down -v

.PHONY: test-clean
test-clean:
	@echo "=== Очистка состояния БД и кеша ==="
	./scripts/test_teardown.sh

.PHONY: test-docker
test-docker:
	@echo "=== Запуск тестов внутри изолированного контейнера test-runner ==="
	docker compose -f docker-compose.test.yml run --rm test-runner

.PHONY: test-integration
test-integration:
	@echo "=== Применение миграций и подготовка БД ==="
	@./scripts/test_init.sh
	@echo "=== Запуск Интеграционных тестов ==="
	go test -v -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration"

.PHONY: test-e2e
test-e2e:
	@echo "=== Запуск Сквозного E2E тестирования (Black-Box через Gateway HTTP API) ==="
	@./scripts/test_stand_up.sh && go test -v -count=1 -tags=e2e ./tests/e2e/... ; STATUS=$$? ; ./scripts/test_stand_down.sh ; exit $$STATUS

.PHONY: capture-traffic
capture-traffic:
	@echo "=== Захват сетевого трафика E2E-тестирования в reports/traffic_capture.pcap ==="
	@./scripts/capture_traffic.sh

# ==============================================================================
# Проверка требований ЛР №2 (№14 и №15)
# ==============================================================================
.PHONY: test-req14
test-req14:
	@echo "======================================================================"
	@echo " [ТРЕБОВАНИЕ 14] Проверка идемпотентности и повторяемости тестов"
	@echo " Запуск интеграционных тестов 3 раза подряд на одном окружении..."
	@echo "======================================================================"
	@docker compose -f docker-compose.test.yml up -d --wait postgres-test redis-test > /dev/null 2>&1 || true
	@./scripts/test_init.sh
	@for i in 1 2 3; do \
		echo ""; \
		echo "--> [Прогон $$i / 3] Выполнение интеграционных тестов..."; \
		go test -v -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration" || exit 1; \
		echo "--> [Прогон $$i / 3] УСПЕШНО пройден!"; \
	done
	@echo ""
	@echo "======================================================================"
	@echo " [ТРЕБОВАНИЕ 14 ВЫПОЛНЕНО]: Интеграционные тесты успешно выполнены"
	@echo " 3 раза подряд без деградации или поломки тестового окружения!"
	@echo "======================================================================"

.PHONY: test-req15
test-req15:
	@echo "======================================================================"
	@echo " [ТРЕБОВАНИЕ 15] Проверка конкурентной изоляции тестов"
	@echo " Одновременный запуск 4 параллельных разработчиков в общей БД..."
	@echo "======================================================================"
	@docker compose -f docker-compose.test.yml up -d --wait postgres-test redis-test > /dev/null 2>&1 || true
	@./scripts/test_init.sh
	@mkdir -p /tmp/req15_logs
	@( \
		echo "--> Запуск потока разработчика #1..."; \
		go test -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration" > /tmp/req15_logs/dev1.log 2>&1 & PID1=$$!; \
		echo "--> Запуск потока разработчика #2..."; \
		go test -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration" > /tmp/req15_logs/dev2.log 2>&1 & PID2=$$!; \
		echo "--> Запуск потока разработчика #3..."; \
		go test -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration" > /tmp/req15_logs/dev3.log 2>&1 & PID3=$$!; \
		echo "--> Запуск потока разработчика #4..."; \
		go test -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration" > /tmp/req15_logs/dev4.log 2>&1 & PID4=$$!; \
		FAIL=0; \
		wait $$PID1 || FAIL=1; \
		wait $$PID2 || FAIL=1; \
		wait $$PID3 || FAIL=1; \
		wait $$PID4 || FAIL=1; \
		if [ $$FAIL -ne 0 ]; then \
			echo "ОШИБКА: Один или несколько параллельных прогонов упали!"; \
			tail -n 20 /tmp/req15_logs/*.log; \
			exit 1; \
		fi \
	)
	@echo ""
	@echo "Логи параллельных прогонов:"
	@head -n 2 /tmp/req15_logs/dev1.log | sed 's/^/  [Dev 1] /'
	@tail -n 1 /tmp/req15_logs/dev1.log | sed 's/^/  [Dev 1] /'
	@head -n 2 /tmp/req15_logs/dev2.log | sed 's/^/  [Dev 2] /'
	@tail -n 1 /tmp/req15_logs/dev2.log | sed 's/^/  [Dev 2] /'
	@head -n 2 /tmp/req15_logs/dev3.log | sed 's/^/  [Dev 3] /'
	@tail -n 1 /tmp/req15_logs/dev3.log | sed 's/^/  [Dev 3] /'
	@head -n 2 /tmp/req15_logs/dev4.log | sed 's/^/  [Dev 4] /'
	@tail -n 1 /tmp/req15_logs/dev4.log | sed 's/^/  [Dev 4] /'
	@echo ""
	@echo "======================================================================"
	@echo " [ТРЕБОВАНИЕ 15 ВЫПОЛНЕНО]: Все 4 параллельных разработчика успешно"
	@echo " завершили прогон тестов без конфликтов и взаимных блокировок!"
	@echo "======================================================================"

.PHONY: test-reqs
test-reqs: test-req14 test-req15

# ЛР №3 WEB:
.PHONY: generate-api
generate-api:
	@echo "=== Кодогенерация REST API v2 из OpenAPI 3.1 (oapi-codegen) ==="
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 -config services/gateway/api_v2_cfg.yaml docs/openapi/openapi.yaml

.PHONY: build-all
build-all:
	@echo "=== Сборка всех микросервисов и CLI ==="
	go build ./services/auth/...
	go build ./services/accounts/...
	go build ./services/transactions/...
	go build ./services/gateway/...
	go build ./tools/cli/...
	@echo "=== Все сервисы успешно собраны! ==="

.PHONY: build-gateway
build-gateway:
	go build -o ./bin/gateway ./services/gateway/cmd/gateway/main.go

.PHONY: run-gateway
run-gateway:
	cd services/gateway && go run ./cmd/gateway/main.go