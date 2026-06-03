# Contributing to Hydra-Pay

## Development Setup

See [docs/DEV_SETUP.md](docs/DEV_SETUP.md) for detailed local environment setup.

Quick start with Docker Compose:

```bash
docker compose up -d
```

This starts PostgreSQL, Redis, pay-service, admin, pay-frontend, portal, Prometheus, and Grafana.

## Code Standards

- **Go**: Follow standard Go conventions. Use `gofmt` before committing.
- **TypeScript/React**: Use the project's existing ESLint/Prettier config.
- **Naming**: Use descriptive names. No single-letter variables except loop indices.
- **Comments**: Write WHY, not WHAT. Code should be self-documenting.
- **Imports**: Group stdlib, third-party, and internal imports with blank lines.

## Testing

```bash
# Unit tests (no Docker required)
cd service && go test ./... -short

# Integration tests (requires Docker)
go test ./internal/integration -v

# Coverage
go test ./... -coverprofile=coverage.out
```

- All new functionality should include tests.
- Integration tests use testcontainers-go and require Docker.
- Test webhook signatures with the provided test helpers in channel packages.

## Pull Request Process

1. Create a branch from `main`.
2. Make your changes with clear, atomic commits.
3. Write or update tests for your changes.
4. Ensure `go test ./... -short` passes.
5. Submit a PR with a clear description of what changed and why.

## Commit Message Convention

```
<type>: <short summary>

<optional body explaining WHY>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`

Examples:

- `feat: add Stripe channel adapter`
- `fix: webhook signature verification for WeChat V3`
- `docs: add API reference for refunds`

## Adding a New Payment Channel

1. Create `service/internal/channel/<channel>/` directory.
2. Implement the `channel.Adapter` interface:
   - `CreatePayment(ctx, req) (*Result, error)`
   - `CreateRefund(ctx, req) (*RefundResult, error)`
   - `VerifyCallback(ctx, data) (*CallbackResult, error)`
   - `QueryPayment(ctx, tradeNo) (*QueryResult, error)`
3. Add a channel constant in `service/internal/model/payment.go` if needed.
4. Register the adapter in `service/cmd/server/main.go` and `admin/handler.go`.
5. Add tests using the pattern in `alipay/alipay_test.go`.
6. Add the channel to `database.seedPaymentChannels()`.

## License

By contributing, you agree that your contributions will be licensed under the project's MIT License.
