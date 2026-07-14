# runx

**Clone. Run. Done.**

`runx` detects your project stack, installs runtimes and dependencies, sets up `.env`, and starts the dev server. Zero config.

```bash
git clone <any-repo>
cd <repo>
runx
```

```
  runx — universal project runner

  → Detecting project stack...
  ✓ Detected node (my-app)

  Runtime
  ✓ Node.js v20.11.0 found

  Dependencies
  ✓ Installing dependencies (pnpm install)...

  Environment
  → Found .env.example, creating .env...
  · PORT → 3000 (auto-filled)
  · DATABASE_URL → postgres://localhost:5432/myapp (auto-filled)
  ? STRIPE_SECRET_KEY: sk-test-...
  ✓ .env created (5 variables)

  Starting
  ─ Running pnpm dev
```

## Install

```bash
# Homebrew
brew install beargruug/tap/runx

# Go
go install github.com/beargruug/runx@latest

# From source
git clone https://github.com/beargruug/runx && cd runx && go build -o runx .
```

## Supported Stacks

| Stack | Marker Files | Package Manager Detection |
|-------|-------------|--------------------------|
| **Node.js** | `package.json` | npm, yarn, pnpm, bun (via lockfile) |
| **Python** | `pyproject.toml`, `requirements.txt`, `setup.py` | pip, poetry, uv, pipenv |
| **Rust** | `Cargo.toml` | cargo |
| **Go** | `go.mod` | go modules |
| **Ruby** | `Gemfile` | bundler (Rails auto-detected) |
| **Docker** | `docker-compose.yml`, `compose.yml` | docker compose |
| **Makefile** | `Makefile` | make |

## Monorepo Support

`runx` detects monorepos automatically:

- **npm/yarn/bun** workspaces (`package.json` → `workspaces`)
- **pnpm** workspaces (`pnpm-workspace.yaml`)
- **Cargo** workspaces (`Cargo.toml` → `[workspace]`)
- **Go** workspaces (`go.work`)

```bash
runx           # interactive picker — choose which package to run
runx web       # run a specific package by name
runx --all     # run all packages in parallel
```

## `.env` Setup

When `runx` finds a `.env.example` (or `.env.sample`, `.env.template`):

1. Copies it to `.env`
2. Auto-fills known defaults (`PORT`, `DATABASE_URL`, `REDIS_URL`, etc.)
3. Prompts you for values that look like placeholders (`YOUR_API_KEY`, `changeme`, etc.)

## Flags

```
runx                  Auto-detect and run
runx <name>           Run a specific monorepo package
runx --all            Run all monorepo packages in parallel
runx --dry-run        Show what would be done without doing it
runx --skip-install   Skip runtime and dependency installation
```

## Runtime Installation

If the required runtime isn't installed, `runx` installs it automatically via [mise](https://mise.jdx.dev) (Rust uses rustup, Docker must be pre-installed).

## License

MIT
