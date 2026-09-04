# AskShop Architecture

## Service Map

```
                                   ┌─────────────┐
                                   │   Client    │
                                   └──────┬──────┘
                                          │ HTTP /api/v1/*
                                          ▼
                                   ┌─────────────┐
                                   │ api-gateway │  :8081
                                   └──────┬──────┘
                    ┌───────────┬─────────┼─────────┬───────────┐
                    │ REST      │ REST    │ REST    │ REST      │ REST
                    ▼           ▼         ▼         ▼           ▼
              ┌──────────┐┌──────────┐┌────────┐┌─────────┐┌──────────┐
              │  user-   ││ product- ││ cart-  ││ order-  ││   ai-    │
              │ service  ││ service  ││service ││ service ││ service  │
              │  :8084   ││  :8082   ││ :8083  ││  :8085  ││  :8086   │
              └────┬─────┘└────┬─────┘└───┬────┘└────┬────┘└────┬─────┘
                   │            │ gRPC:9090│           │ REST     │ REST
                   │            └──────────┤           │          │
                   │                       │◄──────────┘          │
                   │                       │◄─────────────────────┘
                   ▼                       ▼
              ┌─────────────────────────────────┐        ┌──────────┐
              │            Postgres              │        │ RabbitMQ │
              │  (per-service tables, one DB)     │        │ askshop. │
              └─────────────────────────────────┘        │  events  │
                                                            └────┬─────┘
                                                                 │
                                                     order-service publishes
                                                     order.event.placed, and
                                                     runs a demo consumer that
                                                     logs what it receives.
```

- **api-gateway** is the only public entry point. It proxies to each service over plain REST
  (`services/api-gateway/rest/client.go`), unwrapping/rewrapping the shared response envelope.
  It does not own any business logic or state.
- **user-service** owns identity: Supabase-backed register/login, with a local JWT fallback.
  Postgres-backed; no fallback if the DB is unreachable.
- **product-service** owns the catalog (products, images, categories, price, stock). Exposes
  both REST (for the gateway and other services' HTTP clients) and gRPC (`shared/proto/product.proto`,
  used by cart-service for live pricing lookups). Falls back to an in-memory repository if
  Postgres is unreachable — the only service that does.
- **cart-service** owns carts and cart items. Calls product-service's gRPC endpoint on every
  `AddItem`/`ValidateCartForCheckout` call so prices and stock are never stale or hardcoded.
  Requires Postgres to start.
- **order-service** owns orders. On `PlaceOrder` it calls cart-service over REST to validate
  and read the cart, snapshots it into an `Order`, tells cart-service to mark the cart converted,
  and publishes `order.event.placed`. Requires Postgres to start.
- **ai-service** is stateless — no database of its own. It calls product-service (search) and
  cart-service (abandoned carts) over REST, and Gemini over the network. Every route degrades to
  503 if `GEMINI_API_KEY`/`GOOGLE_API_KEY` is unset.
- **auth-service** is currently an unimplemented placeholder (see Roadmap in README.md).

## Request Flow: Checkout

1. Client calls `POST /api/v1/cart/items` (via gateway) → cart-service calls product-service's
   gRPC `GetProduct` to get authoritative name/SKU/price/stock, builds the cart item, persists it.
2. Client calls `POST /api/v1/orders` (via gateway) → order-service:
   a. Calls cart-service's `POST /api/v1/cart/checkout/validate` (which itself re-checks every
      item against product-service for price changes and stock).
   b. If valid, calls cart-service's `GET /api/v1/cart` to read the current contents.
   c. Persists an `Order` + `OrderItem`s as a price/quantity snapshot.
   d. Calls cart-service's `POST /api/v1/cart/checkout/complete` to mark the cart converted.
   e. Publishes `order.event.placed` to RabbitMQ (no-ops with a log line if RabbitMQ isn't
      configured — see `shared/events`).

## Event Flow (RabbitMQ)

`shared/events` wraps a single topic exchange (`askshop.events`), with routing keys declared in
`shared/contracts/amqp.go` (`order.event.placed`, `cart.cmd.*`, `ai.cmd.explain_product`, etc.).
Only `order.event.placed` is wired end-to-end today: order-service publishes it, and runs its own
`order-service.notify-stub` consumer that logs what it receives — a stand-in for a future
dedicated notification service. The other routing keys are declared but not yet published or
consumed by anything.

## Why Prices Are Integers

`Product.PriceCents` and `OrderItem.UnitPriceCents` are `int64` minor-currency-unit fields, not
floats, specifically to avoid rounding drift across service boundaries (product → cart → order,
each doing its own arithmetic). `CartItem.UnitPrice` is still a `float64` (pre-existing scaffolding
this revamp built on) — conversion happens at the cart-service boundary
(`services/cart-service/internal/service/cart_service.go`), so a full float→cents migration of
the cart model is a reasonable future cleanup, not a blocker.

## Known Architectural Gaps

- **No JWT verification at the gateway.** Cart/order identity is currently forwarded via an
  `X-User-ID` header the gateway trusts as-is. Wiring `shared/auth` (or a real auth-service)
  into the gateway's request path is the next auth-hardening step.
- **Cart has a unique index on `user_id`.** Once a cart is marked `converted`, the same user
  can't get a fresh `active` cart without a schema change (a status-scoped unique index, or a
  new cart row per checkout) — not yet hit in practice since carts aren't cleared post-checkout,
  but worth knowing before building repeat-checkout flows.
- **No distributed tracing.** Request IDs propagate service-to-service (see
  `shared/logger`/`shared/response`), but there's no OpenTelemetry/Jaeger-style trace across the
  gateway → cart → product hop chain.
