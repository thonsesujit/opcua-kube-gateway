## What does this PR do?

<!-- A concise description of the change and why it was made. -->

## Related issue

<!-- Fixes #<issue_number> -->

## How was it tested?

- [ ] Unit tests added / updated
- [ ] Tested with the OPC-UA simulator (`docker compose up -d simulator`)
- [ ] Tested against a real OPC-UA server

If tested against real hardware, please note the server type:
<!-- e.g. Siemens S7-1200, Kepware KEPServerEX, Prosys Simulation Server -->

## Checklist

- [ ] `make lint` passes
- [ ] `make test` passes (coverage >= 80%)
- [ ] `make generate` was run if `api/v1alpha1/types.go` was changed
- [ ] CHANGELOG.md updated under `[Unreleased]` (for user-visible changes)
- [ ] Documentation updated if behaviour changed
