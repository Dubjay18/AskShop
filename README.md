# AskShop

AskShop is a Go-based, service-oriented commerce backend that experiments with Supabase authentication, gRPC between services, and an API Gateway front door. The repository is organised as a modular monorepo so individual verticals (user, product, cart, order, AI) can evolve independently while sharing contracts, utilities, and infrastructure scripts.

## Architecture at a Glance
- API Gateway exposes REST endpoints on port 8081 and forwards requests to downstream services with shared tracing, response, and validation helpers.
- User Service provides registration and login workflows that use Supabase Auth in production and fall back to local JWT validation when Supabase is unavailable.
- Product Service exposes a gRPC API (port 9090) with skeletal HTTP handlers. It currently runs with an in-memory repository while the PostgreSQL implementation is being finalised.
- Cart, Order, Auth, and AI services are scaffolded entry points ready for future business logic. The `tools/create_service.go` helper can spin up additional services with the same clean architecture layout.
- Shared packages (`shared/...`) offer common contracts (HTTP, gRPC, AMQP event keys), environment loading, database setup (GORM + PostgreSQL), structured logging, retry helpers, and Supabase client utilities.
- Infrastructure is scripted for either Kubernetes (via Tilt + Docker builds) or manual local execution. Secrets are injected through `.env` files or Kubernetes secrets.

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
ANTHROPIC_API_KEY=<your-anthropic-api-key>
AI_MODEL=claude-opus-5

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

# AI service (HTTP :8086) — optional; set ANTHROPIC_API_KEY to enable it,
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

# Ask the shopping assistant (requires ANTHROPIC_API_KEY on ai-service)
curl -X POST http://localhost:8081/api/v1/ai/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"show me wireless headphones under $50"}'
```
auth-service still ships with a stub `main` function.

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
ai-service (`services/ai-service`) uses the official [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go) (`shared` model configurable via `AI_MODEL`, default `claude-opus-5`) and stays disabled (503 on every AI route, logged once at startup) until `ANTHROPIC_API_KEY` is set — nothing else fails to start because of it.
- **Conversational shopping assistant** (`POST /api/v1/ai/chat`) — a manual tool-use loop (see `internal/service/chat_service.go`) where Claude calls a `search_products` tool backed by product-service's real catalog before answering, so it can't invent products, prices, or stock. Accepts optional `history` for multi-turn conversations.
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
  just inert without `ANTHROPIC_API_KEY`. auth-service is still an empty placeholder.
  Semantic/vector search is a reasonable next step beyond the current keyword search.
- cart-service and order-service require a reachable Postgres to start (no in-memory fallback);
  product-service still degrades to an in-memory repository if Postgres is unreachable.
- The AI cart-nudge endpoint has no scheduler wired up yet — it's designed to be triggered by
  cron/Tilt/an external job runner, not called automatically.
- Tilt pipeline currently builds Linux/amd64 binaries; adjust if your target architecture differs.
- Per-service unit tests are still thin. A GitHub Actions CI workflow (`.github/workflows/ci.yml`)
  now runs `gofmt`, `go vet`, `go build`, and `go test` on every push/PR to `main`.

## Support & Questions
Open an issue or leave notes in the relevant service README. When investigating Supabase connectivity, start with `scripts/diagnose-supabase.sh` and ensure your `.env` matches the latest Supabase dashboard credentials.
