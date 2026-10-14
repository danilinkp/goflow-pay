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
	@rm -rf $(ALLURE_RESULTS) $(ALLURE_REPORT)
	@mkdir -p $(ALLURE_RESULTS)
	@for i in 1 2 3; do \
		echo "=== Прогон $$i из 3 ==="; \
		rm -f $(ALLURE_RESULTS)/*-result.json $(ALLURE_RESULTS)/*-container.json; \
		[ -d $(ALLURE_REPORT)/history ] && cp -r $(ALLURE_REPORT)/history $(ALLURE_RESULTS)/; \
		ALLURE_RESULTS_DIR=$(abspath $(ALLURE_RESULTS)) go test -count=1 -v ./services/transactions/internal/service/... > /dev/null; \
		ALLURE_RESULTS_DIR=$(abspath $(ALLURE_RESULTS)) go test -count=1 -v -run TestAccountService_Classic ./services/accounts/internal/services/... > /dev/null; \
		npx -y allure-commandline generate $(ALLURE_RESULTS) -o $(ALLURE_REPORT) --clean > /dev/null; \
	done
	@echo "Allure-отчет с историей (3 запуска) успешно сгенерирован в: $(ALLURE_REPORT)"

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
	@echo "=== Очистка состояния БД и кеша (Teardown / Rollback) ==="
	./scripts/test_teardown.sh

.PHONY: test-docker
test-docker:
	@echo "=== Запуск тестов внутри изолированного контейнера test-runner ==="
	docker compose -f docker-compose.test.yml run --rm test-runner