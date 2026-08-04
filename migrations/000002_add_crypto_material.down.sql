ALTER TABLE users
    DROP COLUMN kdf_salt,
    DROP COLUMN kdf_time,
    DROP COLUMN kdf_memory_kb,
    DROP COLUMN kdf_threads,
    DROP COLUMN wrapped_dek;