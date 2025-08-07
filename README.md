
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

File migrasi tersedia di folder `migrations/`. Untuk menjalankan migrasi (contohnya PostgreSQL), kamu bisa menggunakan:

```bash
psql -h $DB_HOST -U $DB_USER -d $DB_NAME < migrations/001_create_notes_table.sql
```

Jika ingin otomatisasi migrasi, kamu bisa gunakan tool seperti [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate):

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate -path migrations -database "${DATABASE_URL}" up
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

##  Struktur Model (contoh `pkg/model/note.go`)

```go
type Note struct {
  ID        int       `json:"id" db:"id"`
  Title     string    `json:"title" db:"title"`
  Content   string    `json:"content" db:"content"`
  CreatedAt time.Time `json:"created_at" db:"created_at"`
  UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
```

##  Contoh Handler (Gin)

```go
router.POST("/notes", func(c *gin.Context) {
  var req CreateNoteRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(400, gin.H{"error": err.Error()})
    return
  }
  // simpan ke DB dan respon 201 ...
})
```

##  Tips & Rekomendasi

- Gunakan Gin untuk routing & middleware (logging, recovery, auth, dsb.)
- Pertimbangkan struct `Request` khusus untuk validasi input
- Layered architecture: pisahkan `handlers`, `services/usecases`, `repositories`
- Error handling yang konsisten dan logging
- Tambahkan testing: unit & integration (terutama untuk handler & service)

##  TODO Selanjutnya

- Integrasi JWT untuk authentication
- Middleware untuk auth & rate limiting
- Dockerize aplikasimu (Dockerfile + docker-compose)
- CI/CD pipeline (GitHub Actions)
- Unit tests (handler & service), integration tests dengan database temporary
