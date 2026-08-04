ALTER TABLE users
    ADD COLUMN kdf_salt      BYTEA   NOT NULL DEFAULT '\x00'::bytea,
    ADD COLUMN kdf_time      INT     NOT NULL DEFAULT 1,
    ADD COLUMN kdf_memory_kb INT     NOT NULL DEFAULT 65536,
    ADD COLUMN kdf_threads   SMALLINT NOT NULL DEFAULT 4,
    ADD COLUMN wrapped_dek   BYTEA   NOT NULL DEFAULT '\x00'::bytea;