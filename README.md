# LatihanFiber
REST API sederhana untuk manajemen data mahasiswa 

## Struktur Proyek

```
LatihanFiber/
├── app/
│   ├── model/
│   │   └── student.go        struct entitas, request, dan respons
│   └── repository/
│       └── student_repository.go   kontrak dan implementasi akses data
├── config/
│   └── env.go                 memuat variabel environment
├── database/
│   └── postgres.go            koneksi dan connection pool
├── migrations/
│   └── 001_create_students.sql   skema tabel
├── .env                        konfigurasi lokal (tidak ikut ter-commit)
├── .env.example                 contoh isian variabel environment
├── main.go
├── handler.go
└── helper.go
```

## Skema Tabel

Tabel `students`:

| Kolom        | Tipe            | Keterangan                                      |
|--------------|-----------------|--------------------------------------------------|
| `id`         | `SERIAL`        | Primary key                                      |
| `nim`        | `VARCHAR(20)`   | Nomor Induk Mahasiswa, wajib unik                |
| `name`       | `VARCHAR(100)`  | Nama mahasiswa                                   |
| `grade`      | `DOUBLE PRECISION` | Nilai, default `0`                            |
| `is_active`  | `BOOLEAN`       | Status aktif, default `TRUE`                     |
| `created_at` | `TIMESTAMPTZ`   | Waktu data dibuat, default `NOW()`               |


Skema lengkap tersedia di [`migrations/001_create_students.sql`](migrations/001_create_students.sql).

## Menyiapkan Basis Data dari Nol

Pastikan PostgreSQL sudah terpasang dan service-nya berjalan, lalu:

**1. Buat database kosong**

```bash
psql -U postgres -c "CREATE DATABASE praktikum_backend;"
```

**2. Jalankan migrasi**

```bash
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
```

**3. Verifikasi tabel sudah terbentuk**

```bash
psql -U postgres -d praktikum_backend -c "\d students"
```

Pastikan hasilnya menampilkan kolom `id, nim, name, grade, is_active, created_at` beserta index `students_nim_key` (unique) dan `students_name_lower_idx`

## Konfigurasi Environment

Salin `.env.example` menjadi `.env`, lalu isi sesuai kondisi database lokal Anda:

```bash
cp .env.example .env
```

Variabel yang diperlukan:

| Variabel        | Keterangan                                              | Contoh              |
|-----------------|-----------------------------------------------------------|----------------------|
| `APP_PORT`      | Port tempat server Fiber berjalan                         | `3000`               |
| `DB_HOST`       | Host PostgreSQL                                            | `localhost`           |
| `DB_PORT`       | Port PostgreSQL                                             | `5432`                |
| `DB_USER`       | Username PostgreSQL                                        | `postgres`            |
| `DB_PASSWORD`   | Password PostgreSQL                                         | *(isi milik Anda)*    |
| `DB_NAME`       | Nama database yang dibuat pada langkah setup                | `praktikum_backend`  |
| `DB_SSLMODE`    | Mode SSL koneksi (`disable` untuk pengembangan lokal)        | `disable`             |
| `DB_MAX_CONNS`  | Jumlah maksimum koneksi pada connection pool                | `10`                   |

## Menjalankan Aplikasi

```bash
go mod tidy
go run .
```

Jika berhasil, server berjalan di `http://localhost:3000` (atau sesuai `APP_PORT` yang diisi)

Cek kesehatan server dan koneksi database:

```bash
curl -i http://localhost:3000/api/v1/health
```

Respons `200` dengan `"server dan database berjalan"` menandakan aplikasi sudah tersambung ke PostgreSQL dengan benar

## Status HTTP yang Dikembalikan

| Status | Kondisi                                              |
|--------|--------------------------------------------------------|
| 200    | Permintaan berhasil                                     |
| 201    | Data berhasil dibuat                                     |
| 204    | Data berhasil dihapus                                    |
| 400    | Body request tidak valid / parameter salah              |
| 404    | Data dengan id yang diminta tidak ditemukan              |
| 409    | NIM yang dikirim sudah terdaftar                          |
| 500    | Kesalahan tak terduga pada server                          |
| 503    | Database tidak dapat dihubungi (khusus `/health`)          |
# Kontrak API - Student REST API

Base URL: `http://localhost:3000/api/v1`

## GET /students

- **Parameter:** `page`, `limit` (maks 20), `search`, `sort` (`id`/`name`/`nim`/`grade`), `order` (`asc`/`desc`), `is_active`
- **Contoh body:** –
- **Status:** `200`

**Contoh respons:**
```json
{
  "success": true,
  "message": "daftar student berhasil diambil",
  "data": [
    {
      "id": 1,
      "name": "Sari",
      "nim": "021123456",
      "grade": 80,
      "is_active": true,
      "created_at": "2026-08-26T21:44:28.9761954+07:00"
    }
  ],
  "meta": { "page": 1, "limit": 10, "total": 1, "total_pages": 1 }
}
```

## GET /students/:id

- **Parameter:** `id` (path, angka)
- **Contoh body:** –
- **Status:** `200` / `400` / `404`

**Contoh respons:**
```json
{
  "success": true,
  "message": "student ditemukan",
  "data": {
    "id": 1,
    "name": "Sari",
    "nim": "021123456",
    "grade": 80,
    "is_active": true,
    "created_at": "2026-08-26T21:44:28.9761954+07:00"
  }
}
```

## POST /students

- **Parameter:** –

**Contoh body:**
```json
{ "nim": "2201", "name": "Sari", "grade": 88 }
```

- **Status:** `201` / `409` / `415` / `422`

**Contoh respons:**
```json
{
  "success": true,
  "message": "student berhasil dibuat",
  "data": {
    "id": 2,
    "name": "Dzakki",
    "nim": "434241053",
    "grade": 90,
    "is_active": true,
    "created_at": "2026-08-26T22:08:04.6787466+07:00"
  }
}
```

## PUT /students/:id

- **Parameter:** `id` (path, angka). Semua field wajib diisi.

**Contoh body:**
```json
{ "name": "Sari Baru", "grade": 90, "is_active": false }
```

- **Status:** `200` / `400` / `404` / `415` / `422`

**Contoh respons:**
```json
{
  "success": true,
  "message": "student berhasil diganti seluruhnya",
  "data": {
    "id": 1,
    "name": "Sari Wijaya",
    "nim": "021123456",
    "grade": 0,
    "is_active": false,
    "created_at": "2026-08-26T21:44:28.9761954+07:00"
  }
}
```

## PATCH /students/:id

- **Parameter:** `id` (path, angka). Field opsional.

**Contoh body:**
```json
{ "is_active": true }
```

- **Status:** `200` / `400` / `404` / `415` / `422`

**Contoh respons:**
```json
{
  "success": true,
  "message": "student berhasil diperbarui sebagian",
  "data": {
    "id": 1,
    "name": "Sari Wijaya",
    "nim": "021123456",
    "grade": 95,
    "is_active": false,
    "created_at": "2026-08-26T21:44:28.9761954+07:00"
  }
}
```

## DELETE /students/:id

- **Parameter:** `id` (path, angka)
- **Contoh body:** –
- **Status:** `204` / `400` / `404`
- **Contoh respons:** (tidak ada body)

---

## Contoh Gagal Validasi (422)

```json
{
  "success": false,
  "message": "validasi gagal",
  "errors": {
    "grade": "harus di antara 0 dan 100",
    "name": "wajib diisi",
    "nim": "wajib diisi"
  }
}
```