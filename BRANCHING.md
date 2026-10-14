# Branching strategy

Trunk-based, short-lived branches. Local app only — no release trains.

## Branches

| Branch | Purpose |
|---|---|
| `main` | Always runnable locally (`make api` + `make web`). Protected by “does it boot?” |
| `feature/<slug>` | New capability (import, envelopes, ollama, …). Merge via PR when possible. |
| `fix/<slug>` | Bugfix off `main`. |
| `chore/<slug>` | Tooling, deps, docs, pstack skills. |

## Rules

1. Branch from latest `main`.
2. One concern per branch; keep PRs reviewable in one sitting.
3. Prefer conventional commits: `feat:`, `fix:`, `chore:`, `docs:`.
4. Before merge: API health green, web loads, one smoke path (inbox or ledger).
5. Delete the branch after merge.

## Example

```bash
git checkout main && git pull
git checkout -b feature/ollama-pdf-import
# …work…
git push -u origin HEAD
gh pr create
```
