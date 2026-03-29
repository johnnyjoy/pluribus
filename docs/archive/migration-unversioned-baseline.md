ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Obsolete: unversioned baseline / `schema_migrations`

This document described reconciling **legacy** databases that had been migrated with an old host `psql` loop but lacked a **`schema_migrations`** table.

**Current tree:** there is **no** upgrade path and **no** `schema_migrations` bookkeeping. Control-plane **`app.Boot`** runs embedded `migrations/*.sql` on every startup (idempotent DDL) and requires a **fresh** Postgres (or an empty `public` schema you are willing to treat as greenfield). For anything else, use a dump/restore or a new database volume.

**Pre-release:** there are no shipped releases and no supported “upgrade existing installs” story yet — do **not** treat boot-time SQL as migrating arbitrary legacy databases.

Do not follow the old `INSERT INTO schema_migrations` steps below; they do not apply to the current runner.

---

<details>
<summary>Historical content (do not use)</summary>

The previous version of this file documented creating `schema_migrations` and inserting filenames so reconcile-only mode would skip already-applied files. That mode and table have been removed.

</details>
