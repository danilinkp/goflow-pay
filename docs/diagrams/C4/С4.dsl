workspace "GoFlow Pay" "Процессинговый шлюз для B2B платежей" {

    model {

        guest = person "Гость" "Незарегистрированный пользователь. Может только зарегистрироваться и войти в систему."
        user = person "Пользователь" "Авторизованный представитель компании. Создаёт платежи, просматривает историю и баланс."
        admin = person "Администратор" "Привилегированный пользователь. Управляет компаниями и просматривает аудит-лог."

        smtpSystem = softwareSystem "SMTP-сервер" "Получает и отображает email-уведомления о статусе платежа." "External"

        goflowpay = softwareSystem "GoFlow Pay" "Система для межкорпоративных платежей. Принимает запросы на перевод средств, проверяет баланс, выполняет переводы и уведомляет участников." {

            frontend = container "Frontend SPA" "Веб-интерфейс. Дашборд, список транзакций, форма создания платежа, просмотр счетов." "React, TypeScript" "Web Browser"

            gatewayService = container "Gateway Service" "Единая точка входа, маршрутизирует запросы по gRPC." "Go, REST/HTTP" {
                gwHandler = component "RouterHandler" "Принимает HTTP-запросы, извлекает токен, вызывает AuthClient для валидации." "Go, net/http"
                gwAuthClient = component "AuthGRPCClient" "gRPC-клиент для обращения к Auth Service. Реализует интерфейс TokenValidator." "Go, gRPC"
                gwTxnClient = component "TransactionGRPCClient" "gRPC-клиент для обращения к Transaction Service." "Go, gRPC"
                gwAccClient = component "AccountGRPCClient" "gRPC-клиент для обращения к Account Service." "Go, gRPC"
                gwRedis = component "RateLimiter" "Ограничение частоты запросов через Redis. Реализует интерфейс RateLimiter." "Go, Redis"

                gwHandler -> gwAuthClient "валидирует токен"
                gwHandler -> gwTxnClient "проксирует запрос на платёж"
                gwHandler -> gwAccClient "проксирует запрос баланса"
                gwHandler -> gwRedis "проверяет rate limit"
            }

            authService = container "Auth Service" "Регистрация, аутентификация, выдача и валидация JWT-токенов и API-ключей." "Go, gRPC" {
                authHandler = component "AuthGRPCHandler" "Точка входа gRPC. Принимает запросы на регистрацию и вход." "Go, gRPC"
                authSvc = component "AuthService" "Бизнес-логика: хеширование пароля, генерация JWT, валидация токена. Зависит от интерфейса UserRepository." "Go"
                authUserRepo = component "UserRepository"
                authKeyRepo = component "ApiKeyRepository"
                authRedisRepo = component "TokenBlacklist" "Хранит отозванные токены в Redis." "Go, Redis"
                authDB = component "Auth PostgreSQL" "Хранит данные о пользователе и компаниях." "PostgreSQL" "Database"
                authRedis = component "Auth Redis" "Blacklist отозванных JWT." "Redis" "Database"

                authHandler -> authSvc "вызывает"
                authSvc -> authUserRepo ""
                authSvc -> authKeyRepo ""
                authSvc -> authRedisRepo ""
                authUserRepo -> authDB "читает / пишет"
                authKeyRepo -> authDB "читает / пишет"
                authRedisRepo -> authRedis "читает / пишет"
            }

            transactionService = container "Transaction Service" "Создаёт транзакции, управляет жизненным циклом." "Go, gRPC" {
                txnHandler = component "TransactionGRPCHandler" "gRPC-обработчик. Принимает запрос на создание и получение транзакции." "Go, gRPC"
                txnSvc = component "TransactionService" "Бизнес-логика: проверка идемпотентности, запрос баланса через, создание транзакции." "Go"
                txnAccClient = component "AccountGRPCClient" "gRPC-клиент для запроса баланса в Account Service." "Go, gRPC"
                txnRepo = component "TransactionRepository" "Доступ к данным платежей." "Go, pgx"
                txnDB = component "Transaction PostgreSQL" "Хранит данные транзакций" "PostgreSQL" "Database"

                txnHandler -> txnSvc "вызывает"
                txnSvc -> txnAccClient
                txnSvc -> txnRepo
                txnRepo -> txnDB "читает / пишет"
            }

            accountService = container "Account Service" "Управляет счетами и балансами. Проверяет баланс (gRPC) и выполняет переводы (Kafka)." "Go, gRPC, Kafka" {
                accGrpcHandler = component "AccountGRPCHandler" "gRPC-обработчик. Принимает запрос на получение баланса." "Go, gRPC"
                accKafkaHandler = component "PaymentEventConsumer" "Kafka Consumer. Слушает payment.approved, вызывает бизнес-логику." "Go, Kafka"
                accSvc = component "AccountService" "Бизнес-логика: проверка баланса, идемпотентное списание и зачисление." "Go"
                accRepo = component "AccountRepository" "Доступ к данным счётов." "Go, pgx"
                accDB = component "Account PostgreSQL" "Хранит данные о счетах компаний." "PostgreSQL" "Database"

                accGrpcHandler -> accSvc "вызывает"
                accKafkaHandler -> accSvc "вызывает"
                accSvc -> accRepo "IAccountRepository"
                accRepo -> accDB "читает / пишет"
            }

            notificationService = container "Notification Service" "Kafka Consumer. Отправляет уведомления клиентам." "Go, Kafka" {
                notifConsumer = component "NotificationConsumer" "Kafka Consumer." "Go, Kafka"
                notifSvc = component "NotificationService" "Бизнес-логика: формирование сообщения, защита от дублей." "Go"
                notifRepo = component "NotificationRepository" "Сохраняет факт отправки." "Go, pgx"
                emailSender = component "EmailSender" "Отправляет email через SMTP." "Go"
                notifDB = component "Notification PostgreSQL" "Таблица: notifications." "PostgreSQL" "Database"

                notifConsumer -> notifSvc "вызывает"
                notifSvc -> notifRepo
                notifSvc -> emailSender
                notifRepo -> notifDB "читает / пишет"
            }

            auditService = container "Audit Service" "Kafka Consumer. Записывает полную историю всех событий." "Go, Kafka" {
                auditConsumer = component "AuditConsumer" "Kafka Consumer. Слушает все топики событий." "Go, Kafka"
                auditSvc = component "AuditService" "Бизнес-логика: формирование аудит-записи." "Go"
                auditRepo = component "AuditRepository" "Доступ к данным аудита." "Go, pgx"
                auditDB = component "Audit PostgreSQL" "Хранит данные аудита." "PostgreSQL" "Database"

                auditConsumer -> auditSvc "вызывает"
                auditSvc -> auditRepo
                auditRepo -> auditDB "читает / пишет"
            }

            kafka = container "Apache Kafka" "Очередь сообщенеий статуса платежа." "Apache Kafka" "Queue"

            redis = container "Redis" "" "Redis" "Database"

            frontend -> gatewayService "REST/HTTPS — создание платежей, просмотр истории"
            gatewayService -> authService "gRPC"
            gatewayService -> transactionService "gRPC"
            gatewayService -> accountService "gRPC"
            gatewayService -> redis
            authService -> redis
            transactionService -> accountService "gRPC"
            transactionService -> kafka "публикует: payment.approved, payment.failed"
            accountService -> kafka "публикует: payment.processed"
            kafka -> notificationService "payment.processed, payment.failed"
            kafka -> auditService "payment.approved, payment.processed, payment.failed"
            kafka -> accountService "payment.approved"
        }

        guest -> goflowpay "Регистрируется, входит в систему"
        user -> goflowpay "Создаёт платежи, просматривает историю и баланс"
        admin -> goflowpay "Управляет системой, просматривает аудит-лог"
        goflowpay -> smtpSystem "Отправляет email-уведомления, SMTP"

        guest -> frontend "Открывает в браузере"
        user -> frontend "Использует веб-интерфейс"
        admin -> frontend "Использует веб-интерфейс"

        emailSender -> smtpSystem "Отправляет письмо, SMTP"
        auditConsumer -> kafka "Читает события"
        notifConsumer -> kafka "Читает события"
        accKafkaHandler -> kafka "Читает payment.approved"
    }

    views {
        systemContext goflowpay "L1_SystemContext" "L1 — Контекст системы" {
            include *
        }

        container goflowpay "L2_Containers" "L2 — Контейнеры" {
            include *
        }

        component gatewayService "L3_Gateway" "L3 — Gateway Service" {
            include *
        }
        component authService "L3_Auth" "L3 — Auth Service" {
            include *
        }
        component transactionService "L3_Transaction" "L3 — Transaction Service" {
            include *
        }
        component accountService "L3_Account" "L3 — Account Service" {
            include *
        }
        component notificationService "L3_Notification" "L3 — Notification Service" {
            include *
        }
        component auditService "L3_Audit" "L3 — Audit Service" {
            include *
        }

        styles {
            element "Person" {
                shape Person
                background #1168bd
                color #ffffff
            }
            element "Software System" {
                background #1168bd
                color #ffffff
            }
            element "External" {
                background #666666
                color #ffffff
            }
            element "Container" {
                background #438dd5
                color #ffffff
            }
            element "Component" {
                background #85bbf0
                color #000000
            }
            element "Database" {
                shape Cylinder
                background #438dd5
                color #ffffff
            }
            element "Queue" {
                shape Pipe
                background #e97625
                color #ffffff
            }
            element "Web Browser" {
                shape WebBrowser
                background #438dd5
                color #ffffff
            }
        }
    }
}
