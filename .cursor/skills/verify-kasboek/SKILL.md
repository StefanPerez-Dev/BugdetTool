---
name: verify-kasboek
description: Verify the local Kasboek stack (Postgres, Go API, React, Ollama). Use after changes to apps/api or apps/web, or when asked to prove the app works.
---

# Verify Kasboek

## Launch

1. `docker compose up -d` — wait until `pg_isready` on kasboek-db.
2. `cd apps/api && go run ./cmd/server` — listens on `:8080`.
3. `cd apps/web && npm run dev` — `http://localhost:5173`.
4. Optional: `ollama serve` + model from `.env`.

## Doctor

```bash
curl -s http://localhost:8080/api/health
curl -s http://localhost:8080/api/accounts | head
curl -s 'http://localhost:8080/api/transactions?reviewed=false' | head
```

Expect `database: ok`. `ollama` may be `down` (rules still categorize).

## Drive

1. Open Inbox — confirm or change a category, press Klopt.
2. Kasboek — add `Markt kaas 12,50`, see it in the month.
3. Enveloppen — change Boodschappen budget, reload.
4. Optional: Probeer een voorbeeldimport.

## Evidence

- Health JSON screenshot or curl output.
- Inbox count decreases after confirm.
- New ledger row after quick-add.

## Cleanup

Leave demo data unless the task says to wipe (`DELETE /api/data`).
