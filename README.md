# Kasboek (BugdetTool)

Local-only personal expense tracker. React UI + **Go** API + PostgreSQL 18 + Ollama.

Ported from the Kasboek v2 design prototype (`Persoonlijke uitgavenvolgerapplicatie`): Inbox review, ledger, envelope budgets, bank-statement import, CSV export.

## Backend choice: Go

**Use Go** for this API (not Node/Python), for these reasons:

| Option | Fit for Kasboek |
|---|---|
| **Go (chosen)** | Single binary, excellent Postgres (`pgx`), typed money-in-cents, cheap concurrency for import jobs, Ollama is plain HTTP. Ideal for a forever-local tool. |
| TypeScript (Hono/Fastify) | Fine if you want one language; slightly slower for long-running local services and money correctness. |
| Python (FastAPI) | Nice for OCR/PDF libs; heavier runtime for a simple CRUD + Ollama proxy. |
| Nest/Spring | Overkill for local-only. |

Stack: Go 1.22+ stdlib router, `pgxpool`, SQL migrations, Ollama `/api/generate`.

## Quick start (local only)

```bash
# 1) Database (dedicated port 5433 — does not touch other Postgres on 5432)
docker compose up -d

# 2) Ollama (separate terminal)
ollama serve
# uses OLLAMA_MODEL from .env (default: mistral-nemo:12b if you already have it)
# optional vision for screenshots:
# ollama pull llava

# 3) API
cp .env.example .env
make api                     # http://localhost:8080

# 4) Web
make web                     # http://localhost:5175
```

Health: `make health` → `{ "database": "ok", "ollama": "ok", ... }`

Note: Vite uses **5175** because 5173 is often taken by other local Docker apps. Postgres for Kasboek uses **5433**.

## What works today

- Inbox: review / re-categorize / confirm / discard / keyboard shortcuts
- Kasboek: month navigation, quick-add, search, CSV export, running balance
- Enveloppen: monthly budgets vs spend
- Import: demo import + file upload (vision via Ollama when available; otherwise demo extract fallback)
- Categorization: rule engine first, Ollama for unknowns
- Seed demo data on empty DB

## Layout

```
apps/web   Vite + React + TypeScript
apps/api   Go API
.cursor/   pstack-oriented project rules + verify skill
```

## Branching

See [BRANCHING.md](./BRANCHING.md).

## pstack

Install the Cursor plugin: `/add-plugin pstack`, then `/setup-pstack`.

Important skills for this repo are listed in `.cursor/rules/pstack-for-kasboek.mdc`.
