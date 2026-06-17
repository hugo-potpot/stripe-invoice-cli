# Réécriture de StripeInvoice en Go — Architecture

## Context

Le projet actuel est un outil **batch CLI en Python** (menu interactif `aioconsole`) qui :
1. **Importe** les marchands d'un compte Stripe en appelant les endpoints AJAX du *dashboard* Stripe avec un cookie de session `__Host-session` (pas l'API officielle).
2. **Exporte** les factures PDF de chaque marchand pour un mois donné, les écrit sur disque puis les zippe.
3. **Envoie** par email les ZIP + CSV des nouveaux marchands.
4. Persiste comptes/marchands en MariaDB (SQLAlchemy async), gère une *blacklist* de marchands à exclure.

Problèmes de l'archi actuelle qui motivent la réécriture : logique métier mélangée à l'I/O et au menu dans `app.py`, requêtes HTTP Stripe codées en dur, aucune frontière testable, parallélisme implicite via asyncio difficile à raisonner.

**Objectif** : une version Go avec une architecture **hexagonale légère (ports & adapters)** — logique métier au centre, transports (CLI aujourd'hui, HTTP demain) et infrastructure (Postgres, Stripe, SMTP, fichiers) en périphérie. Choix validés : **CLI Cobra + couche service réutilisable**, **PostgreSQL**, **sqlc**.

---

## Vue d'ensemble — règle de dépendance

Les dépendances pointent **vers l'intérieur**. Le `domain` et le `service` ne connaissent que des **interfaces (ports)** ; les implémentations concrètes (adapters) sont injectées au démarrage.

```
        ┌─────────────────────────────────────────────┐
        │  cmd/stripeinvoice (main) — composition root │  ← câble tout
        └─────────────────────────────────────────────┘
                    │ instancie & injecte
   ┌────────────────┼─────────────────────────────────────┐
   ▼ (transport)    ▼ (cœur)                  ▼ (adapters infra)
 internal/cli   internal/service ──ports──▶ internal/store   (Postgres/sqlc)
 (Cobra)        internal/domain            internal/stripe   (HTTP dashboard)
                                           internal/mailer   (SMTP)
                                           internal/archive  (CSV + ZIP + FS)
```

- **domain** ne dépend de rien (types métier purs).
- **service** dépend de `domain` + des **interfaces** qu'il définit lui-même.
- **adapters** (store, stripe, mailer, archive) dépendent de `domain` et implémentent les interfaces.
- **cli** et **main** dépendent de `service`. Aucun adapter ne dépend d'un autre.

**Pourquoi** : on peut ajouter une API HTTP plus tard en écrivant un nouveau dossier `internal/http` qui appelle les **mêmes services**, sans toucher à la logique métier. Et chaque adapter est remplaçable par un *fake* en test.

---

## Arborescence proposée

```
stripeinvoice/
├── cmd/
│   └── stripeinvoice/
│       └── main.go              # composition root : charge config, ouvre DB, câble adapters→services→CLI
├── internal/
│   ├── config/
│   │   └── config.go            # struct Config + chargement depuis l'env (validation au boot)
│   ├── domain/
│   │   ├── account.go           # type Account
│   │   ├── merchant.go          # type Merchant
│   │   ├── invoice.go           # type Invoice (PDF + métadonnées)
│   │   └── errors.go            # erreurs sentinelles (ErrNotFound, ErrInvalidCookie…)
│   ├── service/
│   │   ├── ports.go             # INTERFACES : Store, StripeClient, Mailer, Archiver
│   │   ├── import.go            # ImportService.ImportMerchants(ctx, accountID, cookie)
│   │   ├── export.go            # ExportService.ExportInvoices(ctx, accountID, period)
│   │   └── mail.go              # MailService.SendMonthly(ctx, period)
│   ├── store/                   # ADAPTER persistence (implémente service.Store)
│   │   ├── migrations/          # *.sql versionnées (goose)
│   │   ├── queries/             # *.sql sources pour sqlc
│   │   ├── gen/                 # code généré par sqlc (ne pas éditer à la main)
│   │   ├── store.go             # pgxpool + wrappers qui mappent gen.* ↔ domain.*
│   │   └── store_test.go
│   ├── stripe/                  # ADAPTER client dashboard Stripe (implémente service.StripeClient)
│   │   ├── client.go            # construction headers/cookies + GET JSON / GET binaire
│   │   ├── endpoints.go         # URLs centralisées (plus de magic strings)
│   │   └── client_test.go       # httptest contre des fixtures
│   ├── mailer/
│   │   └── smtp.go              # ADAPTER SMTP (implémente service.Mailer)
│   ├── archive/
│   │   ├── csv.go               # écriture CSV nouveaux marchands
│   │   └── zip.go               # ADAPTER FS : écrit PDFs + crée le ZIP (implémente service.Archiver)
│   └── platform/
│       └── logger/              # slog configuré (remplace colorlog)
├── assets/                      # sortie : PDFs, ZIP, CSV (dans .gitignore)
├── sqlc.yaml
├── go.mod
├── Makefile                     # migrate, sqlc generate, build, test, lint
└── .env.example
```

**Pourquoi `internal/`** : empêche tout import du code par un module externe — frontière claire, on garde la liberté de refactorer sans casser personne.

---

## Rôle de chaque couche & justification

### `domain/` — le cœur métier
Types purs (`Account`, `Merchant`, `Invoice`) + erreurs sentinelles. **Zéro dépendance** (pas de SQL, pas de HTTP, pas de tags ORM).
**Pourquoi** : le métier ne doit pas changer si on remplace Postgres par autre chose. C'est le seul niveau stable.

### `service/` — orchestration (use cases)
Une fonction par cas d'usage du menu Python actuel :
- `ImportService.ImportMerchants` — remplace `import_merchant_by_session` + `ask_cookie_session`.
- `ExportService.ExportInvoices` — remplace `export_pdfs` (avec parallélisme maîtrisé, voir plus bas).
- `MailService.SendMonthly` — remplace l'option 3.

Le service **définit les interfaces dont il a besoin** (`ports.go`) et reçoit les implémentations par injection.
**Pourquoi** : c'est ici que vit la logique réutilisable. CLI ou HTTP appellent ces méthodes sans rien savoir de l'infra.

### `service/ports.go` — les contrats (clé de la réutilisabilité)
```go
type Store interface {
    UpsertAccountCookie(ctx context.Context, accountID int64, cookie string) (domain.Account, error)
    GetAccount(ctx context.Context, accountID int64) (domain.Account, error)
    InsertMerchantIfNotExist(ctx context.Context, m domain.Merchant) (created bool, err error)
    ListMerchants(ctx context.Context, accountID int64) ([]domain.Merchant, error) // exclut blacklisted
}

type StripeClient interface {
    FetchMerchants(ctx context.Context, cookie string) ([]domain.Merchant, error)
    ListInvoiceDocuments(ctx context.Context, cookie, accountToken string, p domain.Period) ([]domain.Invoice, error)
    DownloadPDF(ctx context.Context, cookie, accountToken, url string) ([]byte, error)
}

type Archiver interface {
    WriteNewMerchantsCSV(period domain.Period, stripeID string, m []domain.Merchant) (string, error)
    WriteInvoice(period domain.Period, accountID int64, inv domain.Invoice, pdf []byte) error
    ZipAccount(period domain.Period, accountID int64) (string, error)
}

type Mailer interface {
    Send(ctx context.Context, attachments []string) error
}
```
**Pourquoi des interfaces définies côté consommateur** (et non côté adapter) : c'est l'idiome Go (« accept interfaces, return structs »). Les tests de service utilisent des fakes triviaux ; aucun mock framework nécessaire.

### `store/` — persistence Postgres via **sqlc** + **pgx**
- Migrations SQL versionnées dans `migrations/` (outil **goose**).
- Requêtes dans `queries/*.sql` → `sqlc generate` produit du code Go typé dans `gen/`.
- `store.go` enveloppe le code généré et **mappe `gen.Merchant` ↔ `domain.Merchant`**, exposant les méthodes de l'interface `service.Store`.

**Pourquoi sqlc + pgx** : SQL explicite et typé à la compilation (pas la « magie » d'un ORM type SQLAlchemy), `pgxpool` gère le pool de connexions (résout proprement le `NullPool` bricolé du Python). Le mapping gen↔domain garde le domaine libre de toute dépendance DB.

### `stripe/` — adapter HTTP dashboard
Reproduit `app/request/stripe.py` + `fetcher.py` : construction des headers (`Cookie: __Host-session=…`, header `stripe-account`), GET JSON et GET binaire (PDF). URLs centralisées dans `endpoints.go`.
**Pourquoi isoler ça** : c'est la partie la plus fragile (dépend du dashboard non documenté de Stripe). Confinée derrière une interface, on la teste avec `net/http/httptest` et on la remplace par l'API officielle Stripe un jour sans toucher au reste.

### `archive/` & `mailer/`
`archive` = écriture PDF/CSV + ZIP (`archive/zip`, `encoding/csv` stdlib). `mailer` = SMTP (`net/smtp` ou `go-mail` pour les pièces jointes MIME, plus simple).
**Pourquoi** : effets de bord fichiers/réseau isolés derrière interfaces → services testables sans écrire sur disque ni envoyer d'email.

### `cli/` — adapter transport (Cobra)
Commandes **fines** qui parsent les flags et appellent un service :
```
stripeinvoice import  --account 42 --cookie "$STRIPE_COOKIE"
stripeinvoice export  --account 42 --period 2026-05
stripeinvoice send    --period 2026-05
```
**Pourquoi des sous-commandes plutôt qu'un menu** : scriptable, automatisable (cron/CI), chaque commande a un code de sortie testable. La logique reste dans `service`, donc la commande ne contient quasi pas de code.

---

## Décisions techniques transverses

| Sujet | Choix | Pourquoi |
|------|-------|----------|
| **Parallélisme** | `golang.org/x/sync/errgroup` + sémaphore (worker pool borné) pour les téléchargements PDF | Remplace asyncio par un parallélisme **explicite et borné** ; on contrôle la concurrence pour ne pas se faire rate-limiter par Stripe. Première vraie amélioration de scalabilité. |
| **Annulation / timeouts** | `context.Context` propagé partout (signature de toutes les méthodes) | Annulation propre (Ctrl-C), timeouts par requête HTTP/DB. |
| **Config** | struct `Config` chargée et **validée au boot** (env, `caarlos0/env` ou `os.Getenv` manuel) | Échec rapide si une variable manque, au lieu d'un crash au milieu d'un batch. |
| **Erreurs** | `fmt.Errorf("...: %w", err)` + erreurs sentinelles dans `domain/errors.go` | Erreurs enveloppées et inspectables (`errors.Is`), pas de `print` silencieux comme en Python. |
| **Logs** | `log/slog` (stdlib, structuré) | Remplace colorlog ; JSON en prod, texte coloré en local. |
| **Secrets** | cookie de session **jamais** codé en dur ; passé par flag/env, stocké chiffré si possible | Le Python avait un Bearer token en dur (`stripe.py`) — à supprimer. |
| **Migrations** | goose, SQL pur dans `store/migrations/` | Versionnage clair du schéma `accounts`/`merchants` (+ colonne `blacklisted`). |
| **Tests** | TDD : tests de service avec fakes, tests d'adapters avec `httptest`/conteneur Postgres | Chaque port est trivial à fausser. |

---

## Stack / dépendances

- `github.com/spf13/cobra` — CLI
- `github.com/jackc/pgx/v5` (+ `pgxpool`) — driver Postgres
- `sqlc` (outil) — génération du code DB ; `github.com/pressly/goose/v3` — migrations
- `golang.org/x/sync/errgroup` — parallélisme borné
- `github.com/wneessen/go-mail` (ou `net/smtp`) — email avec pièces jointes
- stdlib : `log/slog`, `net/http`, `archive/zip`, `encoding/csv`, `context`

---

## Plan d'implémentation (ordre conseillé, TDD)

1. **Squelette** : `go mod init`, arborescence, `Makefile`, `sqlc.yaml`, `.env.example`, `config` + `logger`.
2. **Domain** : types `Account`, `Merchant`, `Invoice`, `Period`, erreurs sentinelles.
3. **Store** : migrations goose (schéma actuel), `queries/*.sql`, `sqlc generate`, wrappers + mapping domain. Tests sur Postgres.
4. **Ports** : `service/ports.go` (interfaces) + fakes pour les tests.
5. **Stripe adapter** : client headers/cookies, GET JSON + binaire, endpoints centralisés. Tests `httptest`.
6. **Archive + Mailer adapters** : CSV/ZIP, SMTP. Tests.
7. **Services** : `import`, `export` (avec errgroup borné), `mail` — pilotés par tests avec fakes.
8. **CLI Cobra** : `import` / `export` / `send`, branchées sur les services.
9. **main.go** : composition root (config → DB → adapters → services → CLI).
10. **Vérification end-to-end** (ci-dessous) + README.

---

## Vérification

- `go build ./...` et `go vet ./...` sans erreur.
- `go test ./...` — services testés via fakes, adapters via `httptest` + Postgres (conteneur ou local).
- `golangci-lint run` propre.
- Migrations : `goose up` puis `goose down` sur une base jetable.
- **Smoke test manuel** contre une vraie session Stripe :
  - `stripeinvoice import --account <id> --cookie <session>` → vérifier insertion en base + CSV généré dans `assets/csv/`.
  - `stripeinvoice export --account <id> --period 2026-05` → vérifier PDFs + ZIP dans `assets/`.
  - `stripeinvoice send --period 2026-05` → vérifier réception de l'email avec pièces jointes.
- Comparer les sorties (noms de fichiers, contenu CSV, structure ZIP) avec celles de la version Python pour garantir la parité fonctionnelle.
