# Git Workflow Rules

## Branch Strategy

- PRs should target the `develop` branch, not `main`
- See wiki for details

## Commit Format

```
<type>(<scope>): <subject>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`

## PR Verification

After merging PRs:
1. Check CI logs
2. Verify E2E tests pass (not SKIP)
3. If skipped, check Secrets configuration
