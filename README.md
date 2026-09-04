# AskShop

AskShop is a Go-based, service-oriented commerce backend that experiments with Supabase authentication, gRPC between services, and an API Gateway front door. The repository is organised as a modular monorepo so individual verticals (user, product, cart, order, AI) can evolve independently while sharing contracts, utilities, and infrastructure scripts.

## Architecture at a Glance
See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full service map and request/event flow diagrams. Short version:
- API Gateway (port 8081) is the only public entry point; it proxies to every backend service over REST and owns no business logic of its own.
- User Service handles registration/login, using Supabase Auth in production with a local JWT fallback.
- Product Service (REST :8082, gRPC :9090) owns the catalog — Postgres-backed with an in-memory fallback if the DB is unreachable.
- Cart Service (:8083) owns carts, pricing every item live against Product Service over gRPC (never hardcoded).
- Order Service (:8085) validates and converts carts into orders, publishing `order.event.placed` over RabbitMQ.
- AI Service (:8086) is a Gemini-backed shopping assistant, product-explanation, and cart-nudge service, grounded in the real catalog via function calling — inert (503) without a `GEMINI_API_KEY`.
- Auth Service is still an unimplemented placeholder; the `tools/create_service.go` helper can scaffold additional services with the same clean-architecture layout.
- Shared packages (`shared/...`) provide common contracts (HTTP/gRPC/AMQP), environment loading, database setup (GORM + PostgreSQL), a RabbitMQ pub/sub wrapper, structured logging, a `/healthz` helper, retry helpers, and Supabase client utilities.
- Infrastructure supports three local-dev paths: Tilt + Kubernetes, `docker compose`, or running each Go binary manually. Secrets are injected through `.env` files or Kubernetes secrets.

## Repository Layout
```
AskShop/
├── api-gateway/                 # Legacy gateway artefacts (deprecated)
├── build/                       # Generated binaries (ignored in git)
├── infra/                       # Dockerfiles, Kubernetes manifests, helper scripts
│   ├── development/docker       # Service Dockerfiles and Windows build helpers
│   └── development/k8s          # Tilt-consumed manifests (deployments, config)
├── scripts/                     # Utility shell scripts for Supabase and secrets
├── services/                    # Go microservices following clean architecture
│   ├── api-gateway              # Production gateway service
│   ├── user-service             # Fully implemented auth + profile endpoints
│   ├── product-service          # gRPC service with in-memory repository stub
│   ├── cart-service             # Scaffold (main entry point only for now)
│   ├── order-service            # Scaffold
│   ├── auth-service             # Placeholder for token verification logic
│   └── ai-service               # Placeholder for AI-powered features
├── shared/                      # Cross-service Go modules (auth, env, db, proto)
├── test/                        # Manual integration checks (`go run ./test/auth_test.go`)
├── Tiltfile                     # Tilt configuration for iterative K8s development
├── AskShop.postman_collection.json   # HTTP workflows for manual testing
└── tools/                       # Helper programs (e.g. new service generator)
```

## Technology Stack
- Go 1.24.x, Gin, GORM, grpc-go, protobuf, Google UUID, go-playground validator.
- Supabase Auth (service-role key) for managed authentication flows.
- PostgreSQL for persistent storage; defaults to `postgres:5432` with auto migrations.
- RabbitMQ event vocabulary defined under `shared/contracts/amqp.go` (publisher/consumer wiring still in progress).
- Tilt + Docker + Kubernetes manifests for local clusters (Minikube, Docker Desktop, or k3d).
- Postman collections for manual HTTP testing.

## Getting Started
### Prerequisites
- go >= 1.24
- Docker and a local Kubernetes cluster (Tilt workflows expect `kubectl` access)
- Tilt CLI (`brew install tilt` or see https://docs.tilt.dev/install.html)
- `protoc` and the Go protobuf toolchain (`protoc-gen-go`, `protoc-gen-go-grpc`) if you need to rebuild gRPC clients
- Optional but recommended: `make`, `kubectl`, and access to a Supabase project

### Install Dependencies
```bash
go mod download
```
If you plan to work on the product gRPC contracts, ensure the plugins above are available and run:
```bash
make product-proto-gen
```

### Configure Environment Variables
All services load configuration from environment variables (with `.env` support via `shared/env`). Example `.env` skeleton:
```env
# Core service wiring
ENV=development
HTTP_ADDR=:8084
USER_SERVICE_URL=http://localhost:8084
PRODUCT_SERVICE_URL=http://localhost:8082
CART_SERVICE_URL=http://localhost:8083
ORDER_SERVICE_URL=http://localhost:8085
PRODUCT_SERVICE_GRPC_ADDR=product-service:9090
CART_SERVICE_GRPC_ADDR=cart-service:9091
ORDER_SERVICE_GRPC_ADDR=order-service:9092
PAYMENT_SERVICE_GRPC_ADDR=payment-service:9093
AUTH_SERVICE_GRPC_ADDR=auth-service:9094

# RabbitMQ (optional — order-service degrades to log-only if unset/unreachable)
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# AI service (optional — endpoints return 503 until this is set)
GEMINI_API_KEY=<your-gemini-api-key>
AI_MODEL=gemini-3.6-flash

# Database (PostgreSQL)
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/askshop?sslmode=disable
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=askshop
DB_SSLMODE=disable

# Supabase auth (replace with values from your project dashboard)
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_KEY=<anon-or-service-role-key>
SUPABASE_KEY_ADMIN=<service-role-key-for-admin-operations>
# Optional: static IP override when DNS issues arise
SUPABASE_IP=<optional-ip-address>
```
Keep real credentials out of version control. The `scripts/update-k8s-secrets.sh` helper can load `.env` values into Kubernetes secrets, and `scripts/update-supabase-secret.sh` shares Supabase keys with remote environments.

### Supabase Diagnostics
Use `scripts/diagnose-supabase.sh` to test DNS resolution and connectivity. Successful runs can optionally append `SUPABASE_IP` to `.env` for environments behind strict firewalls.

## Running the Project
### Option 1: Tilt + Kubernetes (recommended during active development)
1. Ensure a local Kubernetes cluster is running and `kubectl config current-context` points to it.
2. Build required Go binaries: `tilt up`. The Tiltfile compiles each service into `./build/` before the Docker build stage.
3. Tilt will port-forward core services:
   - API Gateway: http://localhost:8081
   - Product Service: http://localhost:8082 (HTTP stub) and gRPC on localhost:9090
   - User Service: http://localhost:8084
4. Inspect logs directly in Tilt or via `kubectl logs deployment/<service>`.
5. Stop with `tilt down` when you are finished.

Update secrets before bootstrapping Tilt when Supabase or database credentials change:
```bash
./scripts/update-k8s-secrets.sh
kubectl rollout restart deployment user-service
```

### Option 2: Run individual services locally
The services are standard Go applications. In separate terminals:
```bash
# Start Postgres and RabbitMQ (or point DB_HOST/RABBITMQ_URL at existing instances)

# User service (HTTP :8084)
go run ./services/user-service/cmd/main.go

# Product service (HTTP :8082, gRPC :9090)
go run ./services/product-service/cmd

# Cart service (HTTP :8083) — requires product-service's gRPC endpoint
go run ./services/cart-service/cmd

# Order service (HTTP :8085) — requires cart-service's HTTP endpoint
go run ./services/order-service/cmd

# AI service (HTTP :8086) — optional; set GEMINI_API_KEY to enable it,
# otherwise its endpoints return 503
go run ./services/ai-service/cmd

# API gateway (HTTP :8081) — fronts all of the above
go run ./services/api-gateway
```
Seed the product catalog for local testing:
```bash
go run ./services/product-service/cmd/seed
```
Then exercise the full commerce flow through the gateway:
```bash
# Register a user (Supabase-powered)
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"Demo123!","first_name":"Demo","last_name":"User"}'

# Log in
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"Demo123!"}'

# Browse products
curl http://localhost:8081/api/v1/products

# Add an item to cart (X-User-ID stands in for JWT-derived identity until
# the gateway forwards it automatically)
curl -X POST http://localhost:8081/api/v1/cart/items \
  -H 'X-User-ID: demo-user' -H 'Content-Type: application/json' \
  -d '{"productId":"<product-id-from-above>","quantity":1}'

# Place an order from the cart
curl -X POST http://localhost:8081/api/v1/orders -H 'X-User-ID: demo-user'

# Ask the shopping assistant (requires GEMINI_API_KEY on ai-service)
curl -X POST http://localhost:8081/api/v1/ai/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"show me wireless headphones under $50"}'
```
auth-service still ships with a stub `main` function.

### Option 3: docker compose (fastest to start, no Tilt/K8s required)
For local iteration without a Kubernetes cluster:
```bash
docker compose up -d postgres rabbitmq          # infra first
docker compose up -d                            # start every service
docker compose run --rm product-service go run ./services/product-service/cmd/seed
```
Each service container runs `go run` against the mounted source tree — no image build step,
edits take effect on `docker compose restart <service>`. Set `GEMINI_API_KEY`, `SUPABASE_URL`,
`SUPABASE_KEY`, and (only if you want to override the `postgres` local-dev default) `POSTGRES_USER`
in a `.env` file at the repo root (docker compose reads it automatically) to enable AI features
and Supabase auth. The compose Postgres container uses trust auth (no password needed or checked)
since it's local-only and never exposed beyond the compose network. Ports match the manual/Tilt
setup: gateway on `:8081`,
product on `:8082`/`:9090` (gRPC), cart on `:8083`, user on `:8084`, order on `:8085`, AI on
`:8086`, Postgres on `:5432`, RabbitMQ on `:5672` (management UI on `:15672`).

## API Surface
The gateway exposes REST resources under `/api/v1` using the contracts defined in `shared/contracts/routes.go`.
- `POST /api/v1/auth/register` – create a user through Supabase Auth.
- `POST /api/v1/auth/login` – email/password login (returns Supabase tokens).
- `GET /api/v1/auth/service-diagnostics` – development-only pass-through to user service diagnostics.
- `GET /api/v1/users/:id` and `GET /api/v1/users/by-email?email=` – fetch user profiles (served by user-service).
- `GET /api/v1/products`, `GET /api/v1/products/:id`, `GET /api/v1/products?q=` – browse and search the catalog (served by product-service).
- `GET /api/v1/cart`, `POST /api/v1/cart/items`, `PUT|DELETE /api/v1/cart/items/:itemId` – manage the caller's cart (served by cart-service, priced live against product-service).
- `POST /api/v1/orders`, `GET /api/v1/orders`, `GET /api/v1/orders/:id` – place and read orders (served by order-service; validates and converts the cart, publishes `order.event.placed`).
- `POST /api/v1/ai/chat` – conversational shopping assistant (served by ai-service; uses a `search_products` tool against product-service so answers are grounded in the real catalog, never invented).
- `POST /api/v1/ai/products/:id/explain` – short AI-generated explanation of a single product.

Use `AskShop.postman_collection.json` with the accompanying environment file to play through end-to-end flows.

## Database & Migrations
`shared/db` encapsulates PostgreSQL connectivity via GORM. Each service provides auto-migrations when the database is reachable: `UserModel` (user-service), `Product`/`ProductImage`/`Category` (product-service), `Cart`/`CartItem`/`SavedItem` (cart-service), `Order`/`OrderItem` (order-service). product-service falls back to an in-memory repository if Postgres is unreachable; cart-service and order-service require Postgres to start.

## AI Features (ai-service)
ai-service (`services/ai-service`) uses the official [Google Gen AI Go SDK](https://github.com/googleapis/go-genai) (Gemini, model configurable via `AI_MODEL`, default `gemini-3.6-flash`) and stays disabled (503 on every AI route, logged once at startup) until `GEMINI_API_KEY` (or `GOOGLE_API_KEY`) is set — nothing else fails to start because of it.
- **Conversational shopping assistant** (`POST /api/v1/ai/chat`) — a manual function-calling loop (see `internal/service/chat_service.go`) where Gemini calls a `search_products` function backed by product-service's real catalog before answering, so it can't invent products, prices, or stock. Accepts optional `history` for multi-turn conversations.
- **Product explanations** (`POST /api/v1/ai/products/:id/explain`) — a single grounded call using only the fetched product's real fields; publishes `ai.cmd.explain_product` over RabbitMQ.
- **Cart-abandonment nudges** (`POST /api/v1/ai/cart-nudges`, internal/batch — not proxied through the gateway) — reads cart-service's `GET /api/v1/cart/admin/abandoned` and generates a short re-engagement message per abandoned cart. Intended to be triggered on a schedule.

## Events (RabbitMQ)
`shared/events` is a thin publisher/consumer over a single `askshop.events` topic exchange, keyed by the routing keys declared in `shared/contracts/amqp.go`. order-service publishes `order.event.placed` after checkout and runs a demonstration consumer (`order-service.notify-stub`) that logs what it receives — a stand-in for a future dedicated notification service. If `RABBITMQ_URL` is unset or the broker is unreachable, publishing/consuming no-ops with a log line rather than failing service startup.

## Protobuf & gRPC
The product service includes a gRPC definition under `shared/proto/product.proto` with generated bindings in the same folder.
- Regenerate bindings with `make product-proto-gen` (requires `protoc` and plugins).
- The gRPC server listens on `PRODUCT_SERVICE_GRPC_ADDR` (default `:9090`). During Tilt runs, port-forwarding exposes it on localhost.

## Logging, Tracing, and Error Handling
- `shared/logger` provides structured logging with context propagation and a Gin middleware that attaches `request_id` metadata. API gateway requests automatically include request IDs in downstream calls.
- `shared/response` normalises HTTP response envelopes (`success`, `error`, `trace_id`) so clients receive consistent payloads.
- `shared/retry` offers exponential backoff helpers for external integrations.
- `shared/health` registers a consistent `GET /healthz` on every service (`{"service": "<name>", "status": "ok"}`), for container orchestrators and manual checks. It doesn't currently check DB/broker connectivity — a liveness check, not a full readiness check.

## Testing
- `go test ./...` runs unit tests for pure domain logic: cart-service's `CartItemBuilder`/`CartManager` (validation, totals, abandonment detection), product-service's slugify/`BeforeCreate` hooks and pagination clamping, and small `shared/` helpers (`contracts.JoinPaths`, `util.GetRandomAvatar`). Repository and HTTP-handler layers are still only covered by manual/Postman testing — no DB-backed or integration tests yet.

## Scripts & Tooling
- `scripts/diagnose-supabase.sh` – interactive DNS and HTTP troubleshooting for Supabase projects.
- `scripts/update-k8s-secrets.sh` – loads `.env` values into Kubernetes secrets and restarts the user service.
- `scripts/update-supabase-secret.sh` – pushes Supabase credentials to remote clusters.
- `scripts/seed-products.sh` – placeholder for product catalogue seeding.
- `tools/create_service.go` – bootstraps a new service following the clean architecture skeleton used across the repo.

## Testing
- Run unit tests with `go test ./...`.
- Validate Supabase integration with `go run ./test/auth_test.go` (requires valid Supabase keys and will create throwaway accounts).
- Postman collections under the repo root cover gateway auth flows.

## Contributing
1. Format code with `go fmt ./...` and run `go test ./...` before pushing.
2. Keep shared contracts backwards compatible; update dependent services together when changing routes or protobuf definitions.
3. Store secrets only in `.env` or secret managers. Do not commit real Supabase keys.
4. When adding a service, prefer `tools/create_service.go -name <service>` to inherit the standard layout and README template.

## Roadmap & Known Gaps
- The core commerce flow now works end to end: browse products → add to cart (priced live from
  product-service) → checkout → order created → cart converted → `order.event.placed` published
  and consumed over RabbitMQ. Verified locally against real Postgres and RabbitMQ instances.
- Product prices are modeled as `priceCents`/`stockQuantity` (integer minor units), not floating
  currency amounts, to avoid rounding drift; the gRPC contract (`shared/proto/product.proto`) and
  domain models were extended accordingly.
- The gateway forwards identity via an `X-User-ID` header for now — there's no JWT verification
  in the request path yet. Wiring `shared/auth` (or auth-service, still a placeholder) through
  the gateway is the next auth-hardening step.
- ai-service now has a real conversational shopping assistant, product explanations, and
  cart-abandonment nudges (see AI Features above), grounded in the catalog via tool use — it's
  just inert without `GEMINI_API_KEY`. auth-service is still an empty placeholder.
  Semantic/vector search is a reasonable next step beyond the current keyword search.
- cart-service and order-service require a reachable Postgres to start (no in-memory fallback);
  product-service still degrades to an in-memory repository if Postgres is unreachable.
- The AI cart-nudge endpoint has no scheduler wired up yet — it's designed to be triggered by
  cron/Tilt/an external job runner, not called automatically.
- **Fixed: Tilt/Kubernetes port config drift.** The container ports in
  `infra/development/k8s/*-deployment.yaml`, the `port_forwards` in `Tiltfile`, and each
  service's actual `HTTP_ADDR` default in code used to disagree for product-service, cart-service,
  ai-service, and order-service — and the API gateway's `HTTP_ADDR` was wired from a configmap key
  (`GATEWAY_HTTP_ADDR`) the code never read, so changing it silently did nothing. All of these are
  now aligned to each service's real default (product-service also gained a `grpc` port/service
  entry for `:9090`, previously not exposed to other pods at all, which would have broken
  cart-service's gRPC pricing lookups in-cluster). The seeder job's `golang:1.21` base image was
  also bumped to `1.24-bookworm` to match `go.mod`'s `go 1.24.2` requirement. This was a static
  consistency fix verified by YAML-parsing every edited manifest — not by deploying to a live
  cluster, since none was available in this environment. Please sanity-check a real `tilt up`
  before depending on it for anything important.
- Tilt pipeline currently builds Linux/amd64 binaries; adjust if your target architecture differs.
- `go test ./...` now covers cart-service and product-service's pure domain logic (see Testing
  above) plus a couple of `shared/` helpers — repository/handler layers and cross-service flows
  are still untested beyond manual/Postman checks. A GitHub Actions CI workflow
  (`.github/workflows/ci.yml`) runs `gofmt`, `go vet`, `go build`, and `go test` on every push/PR
  to `main`.
- Every service now exposes `GET /healthz` (see Logging/Observability above), but it's a liveness
  check only — it doesn't verify DB/broker connectivity, so it can't yet back a real readiness probe.

## Support & Questions
Open an issue or leave notes in the relevant service README. When investigating Supabase connectivity, start with `scripts/diagnose-supabase.sh` and ensure your `.env` matches the latest Supabase dashboard credentials.
