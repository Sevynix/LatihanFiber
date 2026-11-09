CREATE TABLE IF NOT EXISTS supervisors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    quota INTEGER NOT NULL CHECK (quota > 0),
    supervised_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supervisors_supervised_within_quota
        CHECK (supervised_count >= 0 AND supervised_count <= quota)
);

CREATE TABLE IF NOT EXISTS supervisions (
    student_id INTEGER PRIMARY KEY REFERENCES students(id) ON DELETE CASCADE,
    supervisor_id INTEGER NOT NULL REFERENCES supervisors(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS supervisions_supervisor_idx ON supervisions (supervisor_id);

CREATE OR REPLACE FUNCTION supervisions_decrement_count() RETURNS trigger AS $$
BEGIN
    UPDATE supervisors
    SET supervised_count = supervised_count - 1
    WHERE id = OLD.supervisor_id AND supervised_count > 0;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS supervisions_after_delete ON supervisions;
CREATE TRIGGER supervisions_after_delete
    AFTER DELETE ON supervisions
    FOR EACH ROW EXECUTE FUNCTION supervisions_decrement_count();

INSERT INTO supervisors (name, quota)
SELECT v.name, v.quota
FROM (VALUES
    ('Dr. Ahmad Fauzi', 5),
    ('Dr. Siti Rahma', 5),
    ('Dr. Budi Santoso', 4),
    ('Dosen Uji Rebutan', 3)
) AS v(name, quota)
WHERE NOT EXISTS (SELECT 1 FROM supervisors);

INSERT INTO permissions (name, description) VALUES
    ('supervision:manage', 'Menetapkan, mengganti pembimbing TA dan mengubah kuota pembimbing')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'supervision:manage')
ON CONFLICT DO NOTHING;