# Kontrak API - Student REST API

Base URL: http://localhost:3000/api/v1

Format respons semua endpoint sama, ada success, message, data (kalau ada), meta (khusus list), errors (kalau validasi gagal)

## GET /students
Parameter: page, limit (max 20), search, sort (id/name/nim/grade), order (asc/desc), is_active
Contoh body: -
Status: 200
Contoh respons: {"success":true,"message":"daftar student berhasil diambil","data":[{"id":1,"name":"Sari","nim":"021123456","grade":80,"is_active":true,"created_at":"2026-08-26T21:44:28.9761954+07:00"}],"meta":{"page":1,"limit":10,"total":1,"total_pages":1}}

## GET /students/:id
Parameter: id (path, angka)
Contoh body: -
Status: 200/400/404
Contoh respons: {"success":true,"message":"student ditemukan","data":{"id":1,"name":"Sari","nim":"021123456","grade":80,"is_active":true,"created_at":"2026-08-26T21:44:28.9761954+07:00"}}

## POST /students
Parameter: -
Contoh body: {"nim":"2201","name":"Sari","grade":88}
Status: 201/409/415/422
Contoh respons: {"success":true,"message":"student berhasil dibuat","data":{"id":2,"name":"Dzakki","nim":"434241053","grade":90,"is_active":true,"created_at":"2026-08-26T22:08:04.6787466+07:00"}}

## PUT /students/:id
Parameter: id (path, angka), semua field wajib diisi
Contoh body: {"name":"Sari Baru","grade":90,"is_active":false}
Status: 200/400/404/415/422
Contoh respons: {"success":true,"message":"student berhasil diganti seluruhnya","data":{"id":1,"name":"Sari Wijaya","nim":"021123456","grade":0,"is_active":false,"created_at":"2026-08-26T21:44:28.9761954+07:00"}}

## PATCH /students/:id
Parameter: id (path, angka), field opsional
Contoh body: {"is_active":true}
Status: 200/400/404/415/422
Contoh respons: {"success":true,"message":"student berhasil diperbarui sebagian","data":{"id":1,"name":"Sari Wijaya","nim":"021123456","grade":95,"is_active":false,"created_at":"2026-08-26T21:44:28.9761954+07:00"}}

## DELETE /students/:id
Parameter: id (path, angka)
Contoh body: -
Status: 204/400/404
Contoh respons: (tidak ada body)

Contoh kalau gagal validasi (422):
{"success":false,"message":"validasi gagal","errors":{"grade":"harus di antara 0 dan 100","name":"wajib diisi","nim":"wajib diisi"}}