## Summary

- **Closes:** #
- **Specs / Guardrails:** <!-- e.g., FR-01, NFR-01, T-11, ADR-001 -->
- **Changes:**
  -

## Verification

<!-- Paste exact commands executed after your final edit and key output -->

- [ ] `CGO_ENABLED=0 go build ./cmd/...`
- [ ] `go test -short -race ./...` (or via Linux/CI if local Windows lacks `gcc`)
- [ ] `golangci-lint run ./...`

## Checklist

- [ ] Zero secrets, keys, or tokens committed (`gitleaks` clean)
- [ ] Context propagation, explicit error wrapping, and table-driven unit tests included
- [ ] Architecture/ADR/Requirements docs updated if any state transition, API contract, or guardrail changed
