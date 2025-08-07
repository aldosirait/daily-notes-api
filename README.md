
# Daily Notes API

Simple REST API untuk mencatat harian (daily notes) menggunakan **Go**, **Gin**, dan migrasi database sederhana.

##  Fitur
- CRUD Notes (Create, Read, Update, Delete)
- Migrasi database dengan SQL murni
- Struktur modul Go yang jelas

##  Struktur Direktori

```
/
├── cmd/
│   └── server/       ← Entry point aplikasi
├── internal/         ← Business logic (services, usecases)
├── pkg/              ← Model umum & helper (seperti struct, util)
├── migrations/       ← File migrasi database
├── bin/              ← Binary ter­compile (jika ada)
├── go.mod, go.sum    ← Dependency Go
└── .gitignore
```

##  Prasyarat

- [Go](https://go.dev) 1.20+ terinstal
- Database PostgreSQL/MySQL/SQLite (sesuaikan konfigurasi)
- `git` untuk version control

##  Instalasi & Setup

```bash
git clone https://github.com/aldosirait/daily-notes-api.git
cd daily-notes-api
go mod download
```

Salin file konfigurasi lingkungan `.env.example` ke `.env` dan sesuaikan variabel seperti `DB_HOST`, `DB_USER`, `DB_PASS`, `DB_NAME`, dsb.

##  Migrasi Database

File migrasi tersedia di folder `migrations/`. Untuk menjalankan migrasi, kamu bisa menggunakan:

```bash
migrate -path migrations -database "mysql://root@tcp(localhost:3306)/daily_notes" up
```


##  Menjalankan Aplikasi

```bash
cd cmd/server
go run main.go
```

Secara default server akan berjalan di `http://localhost:8080`.

##  Contoh Endpoints

| Metode | URL             | Deskripsi                       |
|--------|------------------|----------------------------------|
| GET    | `/notes`         | Ambil semua notes                |
| GET    | `/notes/:id`     | Ambil note berdasarkan ID        |
| POST   | `/notes`         | Buat note baru                   |
| PUT    | `/notes/:id`     | Update note berdasarkan ID       |
| DELETE | `/notes/:id`     | Hapus note berdasarkan ID        |



##  TODO Selanjutnya

- Penerapan Rate Limiting
- Implementasi Cache Redis
- Penerapan RDBAC
- Testing
