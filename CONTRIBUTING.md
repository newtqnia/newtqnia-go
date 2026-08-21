# Contributing

Requirements: Go 1.22 or newer.

```bash
go fmt ./...
go vet ./...
go test -race -cover ./...
```

Keep the public surface aligned with `openapi/openapi.yaml`. New endpoints should not be synthesized before they exist in the published API contract. Open an issue before making a breaking API change.
