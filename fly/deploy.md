# notes-memory-core-rag — Deployment Notes (Fly.io)

## App Name
`notes-memory-core-rag`

---

## Initial Setup

```bash
flyctl launch
```

**Prompts:**
- **App name:** `notes-memory-core-rag`
- **Postgres:** Yes (required for pgvector)
- **Deploy now:** Yes

This command:

- generated `fly.toml`  
- provisioned a Fly Postgres instance  
- attached Postgres to the app  
- built and deployed the Docker image  
- assigned a public URL  

---

## Fly Postgres Setup (Important)

After launch, Fly shows the Postgres connection info.

### Save it as the production `DATABASE_URL`:

```bash
flyctl secrets set DATABASE_URL="postgres://<user>:<pass>@<host>:<port>/<db>?sslmode=disable"
```

Replace `<user>`, `<pass>`, `<host>`, `<port>`, and `<db>`  
with the credentials displayed during the Fly setup.

---

## Enable pgvector Extension

Connect to the Fly Postg
