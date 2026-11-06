# TripGo Trip Service

HTTP-сервис второй лабораторной работы курса по Go. Сервис создаёт поездки,
возвращает их по идентификатору и завершает активные поездки. Данные хранятся
в PostgreSQL, изменения статуса дополнительно записываются в
`trip_status_history`. Сервис экспортирует метрики и трейсы через OpenTelemetry.

## Требования

- Go 1.27 или новее;
- `tripgoctl`;
- локальное окружение лабораторной работы 2 с PostgreSQL и observability.

## Запуск

Один раз запустите локальный кластер, затем поднимите окружение проекта:

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
```

`tripgoctl environment start` создаёт `.env` с адресом PostgreSQL. `Makefile`
сначала загружает безопасные значения из `.env.example`, а затем применяет
поверх них значения из `.env`. Поэтому адрес базы берётся из актуального
окружения, а добавленные сервисом HTTP-таймауты не теряются при перезапуске
`tripgoctl`.

Примените миграции и запустите сервис:

```bash
make migrate
make run
```

По умолчанию HTTP-сервер доступен на `localhost:8080`.

## HTTP API

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/v1/trips` | Создать поездку |
| `GET` | `/api/v1/trips/{tripId}` | Получить поездку |
| `POST` | `/api/v1/trips/{tripId}/finish` | Завершить поездку |
| `GET` | `/health` | Проверить, что процесс жив |
| `GET` | `/ready` | Проверить доступность PostgreSQL |

Источник истины для HTTP API -
[`contracts/openapi/trip-service.openapi.yaml`](contracts/openapi/trip-service.openapi.yaml).

Пример создания поездки:

```bash
curl -i -X POST http://localhost:8080/api/v1/trips \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
    "start_point": {"latitude": 59.9398, "longitude": 30.3146},
    "end_point": {"latitude": 59.9290, "longitude": 30.3626},
    "price": 1450
  }'
```

## Переменные окружения

Все переменные обязательны. `Makefile` загружает `.env.example`, затем `.env`
и экспортирует итоговые значения процессам `go`, `goose` и `oapi-codegen`.

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | Адрес HTTP-сервера |
| `HTTP_READ_TIMEOUT` | Таймаут чтения запроса |
| `HTTP_READ_HEADER_TIMEOUT` | Таймаут чтения заголовков |
| `HTTP_WRITE_TIMEOUT` | Таймаут записи ответа |
| `HTTP_IDLE_TIMEOUT` | Таймаут keep-alive соединения |
| `LOG_LEVEL` | Уровень логирования |
| `SHUTDOWN_TIMEOUT` | Общий бюджет graceful shutdown |
| `DATABASE_URL` | Строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | Максимальное число соединений пула |
| `DATABASE_MIN_CONNS` | Минимальное число соединений пула |
| `DATABASE_MAX_CONN_LIFETIME` | Максимальное время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | Таймаут подключения к PostgreSQL |
| `DATABASE_QUERY_TIMEOUT` | Таймаут запроса к PostgreSQL |
| `OTEL_SERVICE_NAME` | Имя сервиса в телеметрии |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Адрес OTLP-коллектора |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | Протокол OTLP (`http/protobuf`) |
| `OTEL_RESOURCE_ATTRIBUTES` | Атрибуты ресурса OpenTelemetry |
| `OTEL_TRACES_SAMPLER` | Режим сэмплирования трейсов |

Безопасные примеры значений находятся в [`.env.example`](.env.example).

## Команды

```bash
make generate      # перегенерировать Go-код из OpenAPI
make migrate       # применить все миграции
make migrate-down  # откатить последнюю миграцию
make run           # запустить сервис
make lint          # запустить обязательные линтеры и форматтеры
make test          # дважды запустить unit-тесты с race detector и покрытием
make test-integration # применить миграции и запустить тесты с реальной БД
```

## Архитектура

Зависимости направлены внутрь и вручную связываются в
`internal/container`:

```text
HTTP → usecase → domain
         ↑
repository → PostgreSQL
```

- `internal/domain/trip` содержит модель поездки, правила перехода состояния и
  доменные ошибки. Пакет не зависит от HTTP, PostgreSQL и OpenTelemetry.
- `internal/usecase/trip/<action>` описывает сценарий. Узкие неэкспортируемые
  интерфейсы репозитория, транзакции и наблюдателя объявлены рядом с их
  потребителем в `contract.go`.
- `internal/http/trip/<action>` переводит HTTP-запросы в аргументы usecase.
  Единый маппинг доменных ошибок в RFC 9457 находится в
  `internal/http/domainerror`.
- `internal/repository/trip` реализует хранение доменной модели, а
  `internal/infra/postgres` создаёт пул и менеджер транзакций.
- `internal/infra/telemetry` инициализирует `slog`, OpenTelemetry SDK,
  экспортёры и бизнес-метрики.
- `internal/container` — единственное место создания конкретных реализаций;
  `cmd/service/main.go` только загружает конфигурацию, запускает сервер и
  выполняет graceful shutdown.

Отдельного пакета `model` нет намеренно: между usecase и repository передаётся
доменная `trip.Trip`, а транспортные структуры из OpenAPI остаются только в
HTTP-слое. Это не создаёт вторую модель с теми же полями и не связывает домен с
API.

## Тесты

Доменные правила проверяются table-driven тестами. Usecase и HTTP-хендлеры
тестируются изолированно с `gomock`; моки генерируются директивами
`go:generate` в каждом `contract.go` и коммитятся рядом с тестами.

Интеграционные тесты репозитория имеют build-тег `integration`, используют
PostgreSQL из `tripgoctl` и случайные UUID. Каждый тест удаляет созданную
поездку. Отдельный сценарий намеренно нарушает ограничение таблицы истории и
проверяет, что транзакция откатила первую вставку в `trips`.

## Observability

HTTP middleware выполняются в порядке `Recover → Tracing → Logging → Metrics`.
Логи пишутся через JSON `slog` и содержат `trace_id`/`span_id`. HTTP и SQL
создают спаны, а метрики и трейсы экспортируются в OTLP-коллектор. Сервис
публикует обязательные HTTP/DB-метрики и бизнес-счётчики
`tripgo.trips.created` и `tripgo.trips.completed`.

После `tripgoctl connect` ссылки на Jaeger и Grafana печатаются в терминал. Для
проверки создайте поездку, найдите её HTTP-трейс вместе с вложенным SQL-спаном в
Jaeger, затем найдите `http_server_request_duration` и
`tripgo_trips_created` в Grafana. `trace_id` из JSON-лога должен находить тот же
трейс.

## Принятые решения

### Транзакции

Менеджер транзакций реализует метод `Do(ctx, fn)`. Он открывает транзакцию,
кладёт `pgx.Tx` в производный контекст и вызывает `fn` с этим контекстом.
Успешный вызов завершается `COMMIT`, ошибка или паника - `ROLLBACK`. Вложенный
`Do` находит транзакцию в контексте и переиспользует её.

Репозиторий сам выбирает исполнителя запроса: транзакцию из контекста либо
`pgxpool.Pool`. Сервисный слой не импортирует `pgx` и не передаёт транзакцию
аргументом.

Используется уровень изоляции `Read Committed`. Для этой работы его достаточно:
запрет двух активных поездок обеспечивает уникальный частичный индекс, а
конкурентное завершение защищено условием `status = 'active'` непосредственно
в `UPDATE`.

### Создание поездки

Создание выполняется внутри одного `Do`: сначала строка добавляется в `trips`,
затем начальный переход записывается в `trip_status_history`. Ошибка второй
вставки откатывает обе операции.

Две активные поездки одного водителя запрещает индекс
`trips_one_active_per_driver_idx`. Ошибка PostgreSQL `23505` по этому индексу
преобразуется в доменную ошибку `ErrDriverBusy` и HTTP `409 driver_busy`.

### Завершение поездки

Репозиторий выполняет условный запрос
`UPDATE ... WHERE id = $N AND status = 'active' RETURNING ...`. Поэтому два
параллельных запроса не могут оба завершить поездку и перезаписать
`finished_at`: один получает обновлённую строку, второй - `ErrTripCompleted`.
Обновление поездки и запись перехода `active → completed` выполняются в одной
транзакции.

### Health checks

`/health` всегда отвечает `200` и не обращается к базе. `/ready` выполняет
`Ping` с `DATABASE_QUERY_TIMEOUT`: доступная база даёт `200`, недоступная -
`503`. Обе ручки возвращают `application/json`.

## Проверка

Перед отправкой работы:

```bash
make generate
make lint
make test
make test-integration
```

Также необходимо проверить трейс в Jaeger, метрики в Grafana, переключение
`/ready` при остановке PostgreSQL и graceful shutdown по `SIGTERM`. Скриншоты
Jaeger и Grafana прикладываются к описанию PR.

## Docker

PostgreSQL уже запускается через `tripgoctl`, поэтому Docker Compose запускает
только сервис. `make docker-up` сначала применяет миграции обычной командой
`make migrate`, затем передаёт `DATABASE_URL` из `.env` в контейнер и собирает
его. Имя хоста `localhost` при этом заменяется на `host.docker.internal`, чтобы
контейнер мог подключиться к базе на компьютере:

```bash
make docker-up
curl -i http://localhost:8080/ready
```

Остановить сервисы, сохранив данные PostgreSQL:

```bash
docker compose down
```

Финальный образ собирается на базе `distroless`, не содержит исходники,
Go toolchain и build-кэш и запускается от пользователя `nonroot`. Измеренный
размер образа - **12,3 MiB** (12 927 578 байт). Перепроверить размер можно
командой:

```bash
docker image ls tripgo-trip-service
```
