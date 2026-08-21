# stripe-invoice-go

CLI tool that automates fetching PDF invoices from the Stripe *dashboard* (not the official API) for a set of accounts, and sending them out monthly by email.

## Prerequisites

- Go 1.25+
- PostgreSQL
- [goose](https://github.com/pressly/goose) (migrations)
- [sqlc](https://sqlc.dev) (DB access code generation, only needed if you change queries)

## Configuration

Copy `.env.example` to `.env` and fill in the following variables:

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | yes | PostgreSQL connection string |
| `SMTP_HOST` | yes | SMTP server host (e.g. `smtp.gmail.com`) |
| `SMTP_PORT` | yes | SMTP port (e.g. `587`) |
| `SMTP_USERNAME` | yes | SMTP username |
| `SMTP_PASSWORD` | yes | SMTP password (for Gmail: an [app password](https://myaccount.google.com/apppasswords), not your account password) |
| `SMTP_FROM` | yes | Sender email address |
| `SMTP_TO` | yes | Recipient email address (fixed) |
| `ARCHIVE_DIR` | no | Output folder for PDFs/ZIPs/CSVs (default: `./archive`) |

The program fails fast at startup with a clear error message if a required variable is missing.

### Database

```bash
make migrate-up          # apply migrations
make migrate-status      # show migration status
make migrate-down        # roll back the last migration
make migrate-reset       # start from scratch (⚠️ wipes the database)
make migrate-create name=migration_name
```

## The Stripe session cookie

The commands that talk to Stripe (`import`, `export`) need the **`__Host-session`** dashboard cookie — this is what authenticates you as a user, not an API token.

How to get it:

1. Log in at [dashboard.stripe.com](https://dashboard.stripe.com) in your browser.
2. Open DevTools (`Cmd+Option+I` on Mac) → **Application** tab (Chrome) or **Storage** (Firefox) → **Cookies** → `https://dashboard.stripe.com`.
3. Find the `__Host-session` row and copy its **value** (not the name — just the value, a long string like `keyinfo_live_...`).
4. Pass only that value to the `--cookie` flag on the commands below.

⚠️ This is a **live** session credential — treat it like a password. Never commit it, never paste it in plain text into a shared ticket/chat. It expires after a while; if a command fails with an auth error, grab a fresh value.

## Usage

Commands are run with `go run ./cmd <command> [flags]` (or, after `go build -o stripeinvoice ./cmd`, directly `./stripeinvoice <command> [flags]`).

### Import an account's merchants

Fetches the list of merchants for the Stripe account and inserts them into the database (only new ones are kept), then generates a CSV of the new merchants.

```bash
go run ./cmd import --account-id 42 --cookie "$STRIPE_COOKIE"
```

| Flag | Required | Description |
|---|---|---|
| `--account-id` | yes | Internal (database) ID of the Stripe account |
| `--cookie` | yes | Value of the `__Host-session` cookie (see above) |

### Export a period's invoices

For each merchant on the account, downloads the PDF invoice matching the given period and produces a ZIP.

```bash
go run ./cmd export --account-id 42 --cookie "$STRIPE_COOKIE" --period 2026-05
```

| Flag | Required | Description |
|---|---|---|
| `--account-id` | yes | Internal (database) ID of the Stripe account |
| `--cookie` | yes | Value of the `__Host-session` cookie |
| `--period` | yes | Month in `YYYY-MM` format |

A failure on one merchant (no invoice this month, network error...) doesn't block the export of the others — failures are listed at the end of the command.

### Send invoices by email

Bundles the ZIPs + CSVs of **all** accounts for a given period into a single email, sent to `SMTP_TO`.

```bash
go run ./cmd send --period 2026-05
```

| Flag | Required | Description |
|---|---|---|
| `--period` | yes | Month in `YYYY-MM` format |

Needs neither `--cookie` nor `--account-id`: this command only reads files already produced by `export` (and the CSV produced by `import`) — it never contacts Stripe. If no export exists for any account for that period, no email is sent.

### Typical workflow

```bash
go run ./cmd import --account-id 42 --cookie "$STRIPE_COOKIE"
go run ./cmd export --account-id 42 --cookie "$STRIPE_COOKIE" --period 2026-05
# repeat import/export for each account, then:
go run ./cmd send --period 2026-05
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

Some tests (`internal/stripe`) are integration tests, skipped by default — they need a real session cookie and are run with:

```bash
STRIPE_TEST_COOKIE="..." go test ./internal/stripe/... -v
```