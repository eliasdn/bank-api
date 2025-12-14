## 2024-05-22 - [Database Indexing Strategy]
**Learning:** SQLite supports `DESC` in index definitions, which is crucial for pagination queries that use `ORDER BY created_at DESC`. Without the `DESC` in the index, the database might still need to perform a sort operation or scan the index backwards (which is fast but explicit direction is better).
**Action:** When optimizing "latest items" lists, always prefer composite indexes `(foreign_key, sort_column DESC)` over simple foreign key indexes.
