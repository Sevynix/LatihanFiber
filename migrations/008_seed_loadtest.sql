INSERT INTO users (username, email, password_hash, role)
SELECT 'mhs' || LPAD(g::TEXT, 3, '0'),
       'mhs' || LPAD(g::TEXT, 3, '0') || '@student.unair.ac.id',
       '!', 'student'
FROM generate_series(1, 60) AS g
ON CONFLICT DO NOTHING;