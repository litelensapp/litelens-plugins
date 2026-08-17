# Helm Plugin

The Helm plugin provides integration with Helm package manager for Kubernetes deployments.

**Structure:**

- `helm/` — Contains the Helm plugin (Go backend + TypeScript/React frontend)
  - `helm/internal/` — Go backend services (Helm API client, release management, chart operations)
  - `helm/frontend/` — TypeScript/React frontend package (`@litelens/helm-plugin-frontend`)

**Development:**

Install dependencies:

```bash
pnpm install
```

Build the Helm frontend:

```bash
pnpm build:helm:fe
```

Run tests:

```bash
# Go backend tests
pnpm test:helm:be

# TypeScript frontend tests (requires @litelens/design-system)
pnpm test:helm:fe
```

**Note:** The frontend package depends on `@litelens/design-system`, which is currently maintained in the main litelens application. To work on the frontend locally, ensure you have access to the design-system package or temporarily adjust the workspace configuration.
