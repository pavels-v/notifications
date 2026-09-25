# Notifications Service

## 1. What it does

Sends dunning SMS notifications to loan customers through the MessageBird API.
An operator names a loan and one of three notification types - `reminder`, `dunning` or `termination`.
The service renders the text from a stored template, saves and sends it, and records what the provider answered.

## 2. Configuration and running

### Environment

| Variable | Default |
| --- | --- |
| `HTTP_ADDR` | `:8080` |
| `MESSAGEBIRD_BASE_URL` | required |
| `MESSAGEBIRD_ACCESS_KEY` | required |
| `MESSAGEBIRD_ORIGINATOR` | `ACME` |
| `MESSAGEBIRD_TIMEOUT` | `10s` |
| `DB_HOST` | `localhost` |
| `DB_PORT` | `5432` |
| `DB_USER` | `notifications` |
| `DB_PASSWORD` | `notifications` |
| `DB_NAME` | `notifications` |
| `DB_SSLMODE` | `disable` |
| `DB_CONNECT_TIMEOUT` | `10s` |

`cmd/migrations` reads only the `DB_*` variables.

### Running locally, against a stand-in provider

```bash
cp .env.example .env
make up                              # up db, perform migrations, start service
make seed                            # seed example data
```

### Migrations

`make up` applies migrations, then starts the service.

`make migrate` is used to apply migrations to a database that is already running, for example one started by `make db-up`.

Migrations are applied by `cmd/migrations` - a separate binary meant to run as a Job or an init container before the service.

### Seed data

```bash
make seed
```

Loads the three templates and two demo loans so the database is populated without manual steps.

The demo loans carry phone numbers with the `999` country code, so testing against the real MessageBird will not actually send an SMS.

### Tests

```bash
make test               # unit tests
make test-integration   # integration tests, against a real database
make hooks              # lint before every push
git push --no-verify    # push skipping the hook
```

Tests that need PostgreSQL live in `test/integration` and are controlled by `TEST_DATABASE_URL` with default set to `localhost:5432`.

### Running against MessageBird

Supply the real `MESSAGEBIRD_BASE_URL` (`https://rest.messagebird.com`) and `MESSAGEBIRD_ACCESS_KEY`.

The live API has never been reached, because registering a MessageBird account did not succeed (including the 10 free trial SMS). The client is covered instead by tests against a local server that replays the response shapes MessageBird documents, including error replies.

## 3. Endpoints

[api/v1/v1_swagger.yaml](api/v1/v1_swagger.yaml), served by a running service at `/v1/swagger/index.html`.

There is no delete on either resource. Managing loans and templates goes beyond sending, and is included because editing template texts otherwise means writing SQL by hand.

A template body may use four placeholders, in single braces and `snake_case`: `{credit_number}`, `{full_name}`, `{amount}` and `{due_date}`. Three name a column, and `{amount}` renders the stored minor units and currency together as one readable figure (`amount_minor` column).

## 4. What production would need

The service is small and driven by manual operator actions, and it is built that way deliberately. These are the gaps that would have to close before it ran a real dunning process.

### A durable record of intent

The row is created in `pending` status before the provider is called, so an SMS cannot go out with nothing to show for it.

Production goes one step further, to a **transactional outbox**: the request writes its intent and
returns `202`, and a background dispatcher-worker sends the SMS.

- **For an at-most-once guarantee** - mark the row as attempted before calling the provider and never retry an ambiguous
  outcome.
- **For an at-least-once guarantee** - retry until the provider confirms, and deduplicate with an idempotency key so a
  retry cannot produce a second SMS.
