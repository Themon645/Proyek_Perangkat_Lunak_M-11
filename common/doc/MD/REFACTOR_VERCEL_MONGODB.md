# Dokumentasi Refactoring: Vercel Serverless & Migrasi MongoDB Atlas

Dokumen ini merangkum secara komprehensif seluruh perubahan arsitektur, restrukturisasi kode, migrasi database dari PostgreSQL ke MongoDB Atlas, penanganan error build/runtime di Vercel, hingga strategi manajemen branch pada repositori **Proyek Perangkat Lunak Modul 11 (M-11)**.

---

## 📌 Daftar Isi
1. [Latar Belakang & Tujuan](#1-latar-belakang--tujuan)
2. [Perbandingan Arsitektur: Sebelum vs Sesudah](#2-perbandingan-arsitektur-sebelum-vs-sesudah)
3. [Perubahan Struktur Direktori & File](#3-perubahan-struktur-direktori--file)
4. [Detail Refactor Vercel Serverless](#4-detail-refactor-vercel-serverless)
   - [4.1 Serverless Entrypoint (`api/index.go`)](#41-serverless-entrypoint-apiindexgo)
   - [4.2 Konfigurasi `vercel.json` & Path Rewrite](#42-konfigurasi-verceljson--path-rewrite)
   - [4.3 Penanganan Routing Terpisah (`routes/`) & CORS](#43-penanganan-routing-terpisah-routes--cors)
   - [4.4 Root Standalone Server (`main.go`)](#44-root-standalone-server-maingo)
5. [Detail Migrasi Database: PostgreSQL ke MongoDB Atlas](#5-detail-migrasi-database-postgresql-ke-mongodb-atlas)
   - [5.1 Singleton Connection Pooling (`database/mongo.go`)](#51-singleton-connection-pooling-databasemongogo)
   - [5.2 Model BSON Tagging (`model/`)](#52-model-bson-tagging-model)
   - [5.3 Implementasi MongoDB Repository (`repository/mongo_purchase_order.go`)](#53-implementasi-mongodb-repository-repositorymongo_purchase_ordergo)
   - [5.4 Pembersihan Data Sampah & Seeding Bersih (`database/mongo_seed.go`)](#54-pembersihan-data-sampah--seeding-bersih-databasemongo_seedgo)
6. [Analisis Error & Solusi Selama Deployment Vercel](#6-analisis-error--solusi-selama-deployment-vercel)
7. [Manajemen Branch Git (`main`, `prod`, `local`)](#7-manajemen-branch-git-main-prod-local)
8. [Panduan Konfigurasi & Environment Variables](#8-panduan-konfigurasi--environment-variables)
9. [Verifikasi Pengujian (Testing)](#9-verifikasi-pengujian-testing)

---

## 1. Latar Belakang & Tujuan

Proyek backend Modul 11 awalnya dirancang sebagai aplikasi monolitik Go tradisional yang:
- Berjalan sebagai server HTTP jangka panjang (*long-running process*) pada port lokal (contoh: `:8080`).
- Menggunakan database relasional **PostgreSQL** lokal (`localhost:5432`) dengan ORM **GORM**.
- Mengandalkan server fisik/VM lokal yang tidak cocok untuk arsitektur cloud serverless.

### Tujuan Refactoring:
1. **Deploy ke Vercel Serverless Functions**: Mengubah backend agar dapat berjalan di infrastruktur serverless Vercel dengan konsumsi resource efisien, skalabilitas instan, dan bebas biaya server idle.
2. **Migrasi ke Cloud Database (MongoDB Atlas)**: Menggantikan PostgreSQL lokal dengan MongoDB Atlas cluster cloud berkecepatan tinggi agar backend yang berjalan di cloud Vercel dapat mengakses database secara stabil tanpa batasan network lokal.
3. **Pembersihan & Standardisasi Data**: Menghapus data testing/sampah yang tidak konsisten dan menggantinya dengan dataset awal (10 PO dan 10 Goods Receipt) yang tervalidasi dan mencerminkan alur bisnis pengadaan sesungguhnya.
4. **Isolasi Lingkungan (Branch Isolation)**: Memastikan branch `main` dan `prod` siap untuk deployment cloud, sementara branch `local` tetap aman dipertahankan pada versi sebelumnya.

---

## 2. Perbandingan Arsitektur: Sebelum vs Sesudah

```
+-----------------------------------------------------------------------------------+
|                               SEBELUM REFACTORING                                 |
+-----------------------------------------------------------------------------------+
  Client (FE) ---> [ cmd/server/main.go (Long-running process) ]
                            |
                     (GORM / SQL)
                            v
               [ PostgreSQL Localhost:5432 ] (Crash di Cloud Vercel)
```

```
+-----------------------------------------------------------------------------------+
|                               SESUDAH REFACTORING                                 |
+-----------------------------------------------------------------------------------+
  Client (FE) ---> [ Vercel Edge Network / Rewrites (vercel.json) ]
                            |
                            v
                   [ api/index.go (Serverless Function) ]
                   - package handler
                   - func Handler(w, r)
                   - URL Normalization (x-matched-path)
                   - Global CORS & Preflight OPTIONS (204)
                            |
                            v
                   [ routes/routes.go (ServeMux Router) ]
                            |
                            v
                   [ controller & application layer ]
                            |
                            v
       [ repository/mongo_purchase_order.go (Native Mongo Driver) ]
                            |
                   (Singleton Pool: MaxPoolSize 10)
                            v
            [ MongoDB Atlas Cloud Cluster (Database: procurement) ]
            Collections: purchase_orders, goods_receipts, payment_records
```

---

## 3. Perubahan Struktur Direktori & File

Berikut ringkasan perubahan struktur file di dalam repository:

| File / Folder | Status | Deskripsi Perubahan |
| :--- | :--- | :--- |
| `api/index.go` | **Dibuat Baru** | Entrypoint serverless function untuk Vercel (`package handler`). |
| `vercel.json` | **Dibuat Baru** | Konfigurasi framework & wildcard routing rewrite Vercel. |
| `main.go` (root) | **Dibuat Baru** | Entrypoint ganda: kompatibel dengan Go build Vercel & standalone server lokal. |
| `cmd/server/main.go` | **Dihapus** | Dihapus karena menyebabkan konflik build Vercel yang mendeteksi PostgreSQL lokal. |
| `routes/routes.go` | **Dibuat Baru** | Centralized router, CORS handler, error normalizer, dan root ping `/`. |
| `routes/purchase_order.go` | **Dibuat Baru** | Pemetaan endpoint HTTP Modul 11 ke controller terkait. |
| `database/mongo.go` | **Dibuat Baru** | Inisialisasi thread-safe singleton connection pool ke MongoDB Atlas. |
| `database/mongo_seed.go` | **Dibuat Baru** | Utility untuk membersihkan database dan menyemai 10 dummy PO & GRN bersih. |
| `model/purchase_order.go` | **Diubah** | Ditambahkan struct tag BSON (`bson:"..."`) untuk serialisasi MongoDB. |
| `model/goods_receipt.go` | **Diubah** | Ditambahkan struct tag BSON (`bson:"..."`). |
| `model/payment_record.go` | **Diubah** | Ditambahkan struct tag BSON (`bson:"..."`). |
| `repository/mongo_purchase_order.go` | **Dibuat Baru** | Implementasi penuh 9 metode bisnis PO langsung ke MongoDB Driver. |
| `repository/purchase_order.go` | **Diubah** | Penambahan constructor `NewMongoPurchaseOrderRepository`. |
| `go.mod` & `go.sum` | **Diubah** | Ditambahkan library `go.mongodb.org/mongo-driver`. |

---

## 4. Detail Refactor Vercel Serverless

### 4.1 Serverless Entrypoint (`api/index.go`)
Vercel mewajibkan fungsi Go serverless berada di dalam direktori `api/` dengan nama package `handler` dan mengekspos fungsi signature standar:
```go
package handler

func Handler(w http.ResponseWriter, r *http.Request) {
    // Penanganan request serverless
}
```

**Fitur utama di `api/index.go`:**
1. **Lazy & Non-Blocking Initialization**: Koneksi database MongoDB diinisialisasi melalui `sync.Once` sehingga tidak menyebabkan cold-start timeout atau crash instan jika koneksi sedang *handshake*.
2. **Panic Recovery**: Dilengkapi wrapper `defer func() { if rec := recover(); ... }()` untuk mencegah runtime Vercel berhenti tiba-tiba (`FUNCTION_INVOCATION_FAILED`).
3. **URL Path Normalization**: Mengatasi perubahan URL akibat rewrite Vercel dengan membaca header `x-matched-path` atau fallback ke `r.RequestURI`.

### 4.2 Konfigurasi `vercel.json` & Path Rewrite
Agar seluruh request diarahkan ke satu router utama tanpa memicu error framework:
```json
{
  "version": 2,
  "framework": null,
  "rewrites": [
    {
      "source": "/(.*)",
      "destination": "/api/index.go"
    }
  ]
}
```
> **Catatan Kritis**: `"framework": null` wajib disetel agar Vercel tidak memaksa mode *Go Framework Preset* yang menuntut web server jangka panjang (*long-running process*).

### 4.3 Penanganan Routing Terpisah (`routes/`) & CORS
Semula routing ditempatkan di `api/register.go`. Namun, hal ini menyebabkan build failure karena Vercel mendeteksi beberapa file `package api` dan `package handler` dalam satu folder.
- Seluruh routing dipindahkan ke package terpisah: `routes/`.
- Dilengkapi dengan middleware **CORS Universal**:
  - `Access-Control-Allow-Origin: *` (atau sesuai env `CORS_ORIGIN`).
  - `Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS`.
  - `Access-Control-Allow-Headers: Content-Type, Authorization, X-Requested-With`.
  - Penanganan otomatis `OPTIONS` (HTTP Preflight) dengan return status `204 No Content`.
- Endpoint Root `/`, `/health`, dan `/favicon.ico` disediakan agar saat diakses melalui browser menghasilkan respons JSON `200 OK` (mencegah Vercel 404/500).

### 4.4 Root Standalone Server (`main.go`)
Dibuat file `main.go` di root project yang fleksibel:
- Membaca environment variable `PORT` (default `8080`).
- Memungkinkan developer menjalankan server secara lokal (`go run main.go`) menggunakan basis router dan database yang persis sama dengan yang berjalan di Vercel.

---

## 5. Detail Migrasi Database: PostgreSQL ke MongoDB Atlas

### 5.1 Singleton Connection Pooling (`database/mongo.go`)
Dalam lingkungan serverless, setiap request baru dapat memicu eksekusi function instance baru. Jika koneksi database dibuat berulang-ulang tanpa pembatasan pool, koneksi MongoDB Atlas akan cepat habis (*connection exhaustion*).

Solusi yang diterapkan:
- Menggunakan `sync.Once` untuk memastikan client MongoDB hanya dibuat **satu kali** per worker instance.
- Konfigurasi Connection Pool:
  ```go
  clientOptions := options.Client().
      ApplyURI(uri).
      SetMaxPoolSize(10).
      SetMinPoolSize(0).
      SetMaxConnIdleTime(60 * time.Second).
      SetConnectTimeout(10 * time.Second)
  ```
- Pemeriksaan berkala status koneksi menggunakan `client.Ping(ctx, nil)`.

### 5.2 Model BSON Tagging (`model/`)
Semua struct data utama kini memiliki tag JSON dan BSON agar format penyimpanan di MongoDB rapi dan sesuai konvensi:
- `PurchaseOrder`: Menyimpan `order_number`, `vendor_id`, `status`, `payment_status`, dan nested document `items`.
- `GoodsReceipt`: Menyimpan pencatatan penerimaan barang parsial beserta detail item yang diterima.
- `PaymentRecord`: Menyimpan bukti dan riwayat pembayaran tagihan.

### 5.3 Implementasi MongoDB Repository (`repository/mongo_purchase_order.go`)
Semua method interface `PurchaseOrderRepository` diimplementasikan ulang tanpa ORM relasional, menggunakan native MongoDB Driver:
1. `Create`: Menyimpan dokumen PO baru ke koleksi `purchase_orders`.
2. `Find`: Mencari PO berdasarkan ID numerik atau string.
3. `List`: Menampilkan daftar seluruh PO dengan sorting ID descending.
4. `Receive`: Mengupdate kuantitas item PO yang sudah diterima.
5. `CreateGoodsReceipt`: Menyimpan log penerimaan barang (GRN) ke koleksi `goods_receipts`.
6. `ReceiptHistory`: Menampilkan riwayat GRN berdasarkan PO ID.
7. `Pay`: Mencatat pembayaran tagihan ke koleksi `payment_records` dan mengupdate status PO menjadi `paid`.
8. `Unpaid`: Mengambil PO yang status pembayarannya masih `unpaid`.
9. `Report`: Menghasilkan ringkasan agregasi data pengadaan.

### 5.4 Pembersihan Data Sampah & Seeding Bersih (`database/mongo_seed.go`)
Data uji coba lama yang tidak konsisten dibersihkan dengan utility otomatis:
- `CleanMongo(ctx, db)`: Menghapus seluruh dokumen dari koleksi `purchase_orders`, `goods_receipts`, dan `payment_records`.
- `SeedMongo(ctx, db)`: Memasukkan **10 data PO** dan **10 data Goods Receipt** dengan variasi status realistis:
  - PO-2026-001 (Approved, Unpaid)
  - PO-2026-002 (Received Partial, Unpaid)
  - PO-2026-003 (Received, Paid)
  - PO-2026-004 (Submitted, Unpaid)
  - PO-2026-005 (Received, Unpaid)
  - PO-2026-006 (Approved, Unpaid)
  - PO-2026-007 (Received Partial, Unpaid)
  - PO-2026-008 (Received, Paid)
  - PO-2026-009 (Rejected, Unpaid)
  - PO-2026-010 (Approved, Unpaid)
- **Endpoint Reset Khusus**: Tersedia endpoint `POST /database/reset-and-seed` yang dapat dipanggil sewaktu-waktu untuk mereset database ke kondisi bersih.

---

## 6. Analisis Error & Solusi Selama Deployment Vercel

Selama proses konfigurasi, beberapa kendala umum pada Vercel berhasil diidentifikasi dan diselesaikan:

### Kasus 1: `Expected a long-running server process, but the user server exited during startup: exit status 1`
- **Gejala**: Log runtime Vercel menampilkan pesan error crash saat startup dan gagal terkoneksi ke `localhost:5432`.
- **Penyebab**: Vercel mendeteksi `cmd/server/main.go` yang memanggil PostgreSQL lokal. Di lingkungan Vercel, tidak ada database lokal yang aktif.
- **Solusi**: Mengalihkan seluruh koneksi ke MongoDB Atlas via `MONGODB_URI` dan menghapus entrypoint lama yang mengikat ke localhost.

### Kasus 2: `Error: [BUILD_FAILED] No Go entrypoint found`
- **Gejala**: Build Vercel gagal dengan pesan: `Expected one of: main.go, cmd/api/main.go, cmd/server/main.go`.
- **Penyebab**: Pengaturan *Framework Preset* di Vercel Dashboard disetel ke **Go**, sehingga Vercel mengharapkan file server konvensional daripada Serverless Function.
- **Solusi**: 
  1. Menambahkan file root `main.go` agar build Go Vercel tetap valid.
  2. Menyetel `"framework": null` pada `vercel.json`.

### Kasus 3: `Multiple packages in directory api/`
- **Gejala**: Compiler Go menolak build karena terdapat package `api` dan package `handler` di folder `api/`.
- **Penyebab**: Penempatan file router tambahan di dalam folder `api/`.
- **Solusi**: Memisahkan semua logika router ke package `routes/` dan menyisakan hanya satu file `api/index.go` dengan `package handler`.

### Kasus 4: Vercel Internal Rewrite Path Mismatch
- **Gejala**: Endpoint seperti `/purchase-orders` menghasilkan status 404 ketika diakses via domain Vercel.
- **Penyebab**: Rewrite internal Vercel mengubah `r.URL.Path` menjadi `/api/index.go`.
- **Solusi**: Mengimplementasikan pengecekan header `x-matched-path` dan parsing `r.RequestURI` di layer routing untuk mengembalikan URL ke path aslinya sebelum dicocokkan oleh router.

### Kasus 5: `mkdir storage: read-only file system` saat Upload Bukti Pembayaran
- **Gejala**: Saat pengguna mengunggah bukti transfer melalui modal pelunasan (`POST /purchase-orders/:id/payment`), serverless mengembalikan HTTP 400 dengan pesan `INVALID_PAYMENT_PROOF: mkdir storage: read-only file system`.
- **Penyebab**: Di lingkungan Serverless Vercel (container AWS Lambda), filesystem root (`/var/task`) berstatus *read-only*. Pemanggilan `os.MkdirAll("storage/payment-proofs")` ditolak oleh sistem operasi dengan kode `EROFS`.
- **Solusi**: 
  1. Menambahkan fungsi `GetProofStorageDir()` di `common/helper/storage.go` dengan fallback otomatis ke `os.TempDir()` (`/tmp` di Linux/Vercel) yang dijamin memiliki izin tulis disk.
  2. Menambahkan fungsi `DeletePaymentProof()` untuk menangani penghapusan file baik di storage lokal maupun fallback `/tmp`.
  3. Menyediakan route statis `GET /storage/payment-proofs/{filename}` pada router untuk melayani file bukti jika diakses kembali.

---

## 7. Manajemen Branch Git (`main`, `prod`, `local`)

Untuk menjaga stabilitas kode dan kemudahan pengembangan, cabang repository dibagi sebagai berikut:

1. **Branch `main` (Remote & Local)**:
   - Branch aktif untuk deployment otomatis ke Vercel (Production/Preview).
   - Memuat seluruh implementasi Serverless (`api/index.go`, `vercel.json`), router CORS, dan integrasi penuh MongoDB Atlas.
2. **Branch `prod` (Remote & Local)**:
   - Branch stabil production yang disinkronkan dengan `main`.
3. **Branch `local` (Local)**:
   - Dikembalikan (*reverted*) ke commit `604fbb8441011821e45cedf8b1b84f68b04b5c18` sesuai permintaan, untuk menjaga catatan histori pekerjaan lokal sebelumnya.

---

## 8. Panduan Konfigurasi & Environment Variables

Untuk menjalankan proyek di Vercel atau secara lokal, pastikan variabel lingkungan berikut telah dikonfigurasi:

### Variabel Lingkungan di Vercel Dashboard (Project Settings > Environment Variables):
```env
# Koneksi MongoDB Atlas Cloud
MONGODB_URI=mongodb+srv://<username>:<password>@hilman.cctd8il.mongodb.net/?retryWrites=true&w=majority&appName=Hilman
MONGODB_DATABASE=procurement

# Konfigurasi CORS (Origin URL Frontend Anda)
CORS_ORIGIN=*
```

> **Penting untuk MongoDB Atlas**:
> Pastikan pada menu **Network Access** di MongoDB Atlas telah ditambahkan IP `0.0.0.0/0` (Allow Access from Anywhere) karena Vercel Serverless Functions menggunakan IP dinamis saat melakukan eksekusi.

---

## 9. Verifikasi Pengujian (Testing)

Seluruh pengujian unit dan integrasi telah diverifikasi dan berjalan dengan status **PASS**:

1. **Uji Koneksi MongoDB Atlas Live**:
   ```bash
   go test -v -run TestLiveMongoConnection ./database/...
   # Output: PASS (Koneksi ke Atlas dan ping database berhasil)
   ```

2. **Uji Alur Bisnis Purchase Order di MongoDB**:
   ```bash
   go test -v -run TestMongoPurchaseOrderFlow ./repository/...
   # Output: PASS (Create PO, Find, Receive, Goods Receipt, Payment, dan Aggregation Report valid)
   ```

3. **Uji Routing & CORS**:
   ```bash
   go test -v ./routes/...
   # Output: PASS (Handling preflight OPTIONS 204 dan path rewrite normalization valid)
   ```

---

*Dokumentasi ini disusun secara otomatis sebagai catatan teknis resmi refactoring Vercel Serverless dan migrasi MongoDB Atlas pada repositori Proyek Perangkat Lunak Modul 11.*
