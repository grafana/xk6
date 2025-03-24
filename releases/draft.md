**xk6** `vx.y.z` is here! 🎉
 
This release includes:
- New golangci-lint configuration

## New features

### New golangci-lint configuration [#149](https://github.com/grafana/xk6/issues/149)

The previous linter rules were taken from `k6` and contained too many k6-specific exceptions and settings.

Principles taken into account when creating the configuration:
- All linter rules should be enabled by default.
- Linter rules should only be disabled in justified cases (personal taste is not a reason to disable a rule).
- Disabling a rule should be justified in a comment.

The xk6 source code has be modified to comply with the new rules.
