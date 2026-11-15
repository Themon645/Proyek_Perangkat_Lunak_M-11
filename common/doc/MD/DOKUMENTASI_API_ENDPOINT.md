# 3.5 API dan Kontrak Antarmodul
**Dokumen Teknis Modul 11 (Pembelian / Procurement) — Araya Business Hub Platform**  
*Format tabel disesuaikan secara presisi dengan standar template Dokumen Teknis (Bab 3.5 `h.o2m6uc2xae5h`) dan spesifikasi [`openapi.yaml`](file:///d:/SEMESTER%207/PPL/Proyek_Perangkat_Lunak_M-11/openapi.yaml).*

---

### Informasi Spesifikasi API

| Item | Isi |
| :--- | :--- |
| **File spesifikasi** | `https://github.com/Themon645/Proyek_Perangkat_Lunak_M-11/blob/main/openapi.yaml` |
| **Halaman dokumentasi** | `https://proyek-perangkat-lunak-m-11.vercel.app/api-docs` (Production Serverless)<br>`http://localhost:8080/api-docs` (Lingkungan Lokal) |

---

### Ringkasan Endpoint

| Endpoint | Method | Autentikasi | Respons berhasil | Respons gagal | User Story · AC |
| :--- | :---: | :--- | :--- | :--- | :--- |
| `/purchase-orders/approved` | `GET` | Bearer token (Administrator) | `200` + array daftar PO disetujui beserta rincian item | `500` “Internal server error” | `M11-US1` · AC-1, AC-2, AC-3<br>`M11-US2` · AC-1 |
| `/goods-receipts` | `POST` | Bearer token (Administrator) | `201` + receipt_no, purchase_order_id, dan items GRN | `400` “Field wajib belum lengkap”<br>`404` “PO tidak ditemukan”<br>`409` “PO belum disetujui / status tidak valid”<br>`422` “Kuantitas melebihi sisa pesanan (Zero Tolerance)” | `M11-US1` · AC-1, AC-2, AC-3, AC-4, AC-5<br>`M11-US2` · AC-1, AC-2, AC-3, AC-4 |
| `/goods-receipts` | `GET` | Bearer token (Administrator) | `200` + array riwayat dokumen GRN & rincian surat jalan | `500` “Gagal mengambil data goods receipts” | `M11-US1` · AC-1<br>`M11-US2` · AC-1, AC-2 |
| `/purchase-orders/{id}/receipts` | `GET` | Bearer token (Administrator) | `200` + riwayat multi-receipt, tanggal, penerima, dan item | `404` “Purchase order tidak ditemukan”<br>`500` “Internal server error” | `M11-US2` · AC-1, AC-2 |
| `/purchase-orders/{id}/payment` | `POST` | Bearer token (Administrator) | `200` + data PO dengan status payment `paid` & `proof_url` Neon S3 | `400` “Format bukti harus PDF/JPG/PNG maks 5 MB”<br>`404` “PO tidak ditemukan”<br>`409` “PO sudah dibayar / barang belum diterima” | `M11-US3` · AC-1, AC-2, AC-3 |
| `/purchase-orders/unpaid` | `GET` | Bearer token (Administrator) | `200` + array purchase_orders belum lunas & total kewajiban bayar | `500` “Database query failed” | `M11-US3` · AC-1 |
| `/purchase-orders` | `GET` | Bearer token (Administrator) | `200` + array seluruh PO, rincian item, kuantitas diterima, dan total biaya | `500` “Database connection timeout” | `M11-US1` · AC-1<br>`M11-US2` · AC-1<br>`M11-US3` · AC-1 |
| `/purchase-orders/{id}` | `GET` | Bearer token (Administrator) | `200` + detail lengkap satu PO, item pesanan, dan status pelunasan | `404` “Purchase order tidak ditemukan”<br>`500` “Internal server error” | `M11-US1` · AC-1<br>`M11-US2` · AC-1 |
| `/purchase-orders` | `POST` | Bearer token (Administrator) | `201` + objek Purchase Order baru | `400` “Supplier dan items wajib diisi”<br>`409` “Purchase order already exists” | `M11-US1` · AC-1 |
| `/purchase-orders/{id}/receipts` | `POST` | Bearer token (Administrator) | `200` + PO dengan pembaruan kuantitas diterima & status baru | `400` “Input tidak valid”<br>`404` “PO tidak ditemukan”<br>`409` “PO belum disetujui / over-delivery” | `M11-US1` · AC-1<br>`M11-US2` · AC-1, AC-3 |
| `/reports/procurement` | `GET` | Bearer token (Administrator) | `200` + rentang periode from-to & rekapitulasi nominal belanja per vendor | `400` “Parameter from dan to (YYYY-MM-DD) wajib diisi”<br>`500` “Database aggregation error” | `M11-US1` · AC-1<br>`M11-US2` · AC-1<br>`M11-US3` · AC-1 |
| `/health` | `GET` | - (Publik) | `200` + status serverless: running & postgres: connected (Neon) | - | `NFR-01` (Keandalan Sistem) |
| `/` | `GET` | - (Publik) | `200` + welcome message & daftar rute API serverless | - | Infrastruktur Runtime |
| `/database/reset-and-seed` | `POST` | Internal Dev | `200` + pesan konfirmasi database berhasil direset & diisi 10 dummy data | `500` “Failed to reset database” | Utilitas Pengujian (Testing Environment) |

---

### Kontrak Antarmodul (Sesuai Bab 2.3)

| Kontrak | Penyedia → Pemakai | Endpoint / Event | Data | Disepakati |
| :--- | :--- | :--- | :--- | :--- |
| `purchase_order.received` | **M11 (Pembelian)** → **M9 (Pengadaan)** | `POST ${M9_GRN_ENDPOINT}` *(Event Webhook via background queue)* | Payload dokumen GRN lengkap (`receipt_no`, `purchase_order_id`, `received_date`, daftar item yang diterima) | 09 November 2026 |

---

## Rincian Spesifikasi Tiap Endpoint

### 1. `GET /purchase-orders/approved`
* **Method**: `GET`
* **Autentikasi**: Bearer token (Administrator)
* **Deskripsi**: Mengambil seluruh PO yang berstatus disetujui (`status = "approved"`) yang siap dipilih pada dropdown formulir penerimaan barang (GRN).
* **Respons Berhasil (`200 OK`)**:
  ```json
  [
    {
      "id": "PO-2026-001",
      "supplier": "PT Graha Sarana Mitra",
      "status": "approved",
      "payment": "unpaid",
      "total": 6500000,
      "items": [
        {
          "id": "ITEM-101",
          "name": "Kertas HVS A4 80gr Sinar Dunia",
          "quantity": 100,
          "unit_cost": 65000,
          "qty_received": 0
        }
      ]
    }
  ]
  ```
* **Respons Gagal (`500 Internal Server Error`)**:
  ```json
  {
    "error": "internal server error"
  }
  ```
* **User Story · AC**:
  * **`M11-US1` · AC-1, AC-2, AC-3**: Hanya PO yang berstatus disetujui yang dapat diproses penerimaan barangnya.
  * **`M11-US2` · AC-1**: Memastikan dasar kuantitas awal pesanan valid sebelum dicatat penerimaan bertahap.

---

### 2. `POST /goods-receipts`
* **Method**: `POST`
* **Autentikasi**: Bearer token (Administrator)
* **Header**: `Content-Type: application/json`
* **Deskripsi**: Mencatat dokumen fisik surat jalan penerimaan barang dari vendor, memvalidasi sisa kuantitas pesanan secara atomik, dan memperbarui status PO (`received partial` atau `received`).
* **Request Body**:
  ```json
  {
    "purchase_order_id": "PO-2026-001",
    "receipt_no": "SJ-2026/XI/1042",
    "received_date": "2026-11-15",
    "receiver": "Petugas Gudang BPU",
    "notes": "Barang diterima dalam kondisi baik dan tersegel",
    "items": [
      {
        "item_id": "ITEM-101",
        "qty_received": 50
      }
    ]
  }
  ```
* **Respons Berhasil (`201 Created`)**:
  ```json
  {
    "receipt_no": "SJ-2026/XI/1042",
    "purchase_order_id": "PO-2026-001",
    "received_date": "2026-11-15T00:00:00Z",
    "receiver": "Petugas Gudang BPU",
    "notes": "Barang diterima dalam kondisi baik dan tersegel",
    "items": [
      {
        "id": "GRN-ITEM-001",
        "receipt_no": "SJ-2026/XI/1042",
        "item_id": "ITEM-101",
        "qty_received": 50
      }
    ]
  }
  ```
* **Respons Gagal**:
  * `400 Bad Request`: `{"error": "purchase_order_id, receipt_no, received_date, and items are required"}`
  * `404 Not Found`: `{"error": "purchase order not found"}`
  * `409 Conflict`: `{"error": "purchase order must be approved before goods receipt"}`
  * `422 Unprocessable Entity`: `{"code": "QUANTITY_EXCEEDS_REMAINING", "message": "kuantitas diterima (150) melebihi sisa pesanan (100) untuk item ITEM-101"}`
* **User Story · AC**:
  * **`M11-US1` · AC-1, AC-3, AC-4, AC-5**: Validasi jumlah pesanan penuh, penolakan input nol/negatif, dan pemicu webhook ke M9.
  * **`M11-US2` · AC-1, AC-2, AC-3, AC-4**: Pencatatan bertahap, kebijakan Zero Tolerance (penolakan over-delivery), dan update status `received partial`.

---

### 3. `GET /goods-receipts`
* **Method**: `GET`
* **Autentikasi**: Bearer token (Administrator)
* **Query Parameter**: `purchase_order_id` *(string, opsional)*, `po` *(string, opsional)*
* **Deskripsi**: Menampilkan seluruh arsip dokumen penerimaan barang fisik yang masuk ke gudang BPU.
* **Respons Berhasil (`200 OK`)**:
  ```json
  [
    {
      "receipt_no": "SJ-2026/X/9814",
      "purchase_order_id": "PO-2026-003",
      "received_date": "2026-11-15",
      "receiver": "Tyezar Hasan",
      "notes": "Barang diterima dalam kondisi baik",
      "items": [
        {
          "item_id": "ITEM-103",
          "item_name": "Kabel LAN Cat6 305 Meter",
          "qty_received": 10
        }
      ]
    }
  ]
  ```
* **Respons Gagal (`500 Internal Server Error`)**:
  ```json
  {
    "error": "failed to fetch goods receipts"
  }
  ```
* **User Story · AC**:
  * **`M11-US1` · AC-1** & **`M11-US2` · AC-1, AC-2**: Bukti fisik penerimaan barang tercatat di database dan dapat ditinjau kembali.

---

### 4. `GET /purchase-orders/{id}/receipts`
* **Method**: `GET`
* **Autentikasi**: Bearer token (Administrator)
* **Parameter Path**: `id` *(string, wajib)* — ID Purchase Order, contoh: `PO-2026-003`
* **Deskripsi**: Menampilkan riwayat multi-receipt per tanggal pengiriman untuk memantau tahapan kedatangan barang bertahap pada satu PO tertentu.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "purchase_order_id": "PO-2026-003",
    "receipts": [
      {
        "receipt_no": "SJ-2026/X/9814",
        "received_date": "2026-11-15",
        "receiver": "Tyezar Hasan",
        "notes": "Barang diterima dalam kondisi baik",
        "items": [
          {
            "item_id": "ITEM-103",
            "item_name": "Kabel LAN Cat6 305 Meter",
            "qty_received": 10
          }
        ]
      }
    ]
  }
  ```
* **Respons Gagal**:
  * `404 Not Found`: `{"error": "purchase order not found"}`
  * `500 Server Error`: `{"error": "database error"}`
* **User Story · AC**:
  * **`M11-US2` · AC-1, AC-2**: Melacak riwayat surat jalan parsial dan sisa barang yang belum terkirim.

---

### 5. `POST /purchase-orders/{id}/payment`
* **Method**: `POST`
* **Autentikasi**: Bearer token (Administrator / Keuangan)
* **Content-Type**: `multipart/form-data`
* **Form Field**: `proof` *(file binary, wajib — PDF/JPG/PNG maks 5 MB)*
* **Deskripsi**: Mengonfirmasi pembayaran tagihan PO yang barangnya sudah diterima (`received` atau `received partial`), mengunggah bukti transfer fisik langsung ke Neon S3 Object Storage, dan mengubah status pembayaran menjadi `paid`.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "id": "PO-2026-005",
    "supplier": "PT Pasundan Niaga Logistik",
    "status": "received",
    "payment": "paid",
    "proof": "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf",
    "proof_url": "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf",
    "attachment_url": "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample-receipt.pdf",
    "total": 27750000,
    "items": [
      {
        "id": "ITEM-105",
        "name": "Kursi Kantor Ergonomis",
        "quantity": 15,
        "unit_cost": 1850000,
        "qty_received": 15
      }
    ]
  }
  ```
* **Respons Gagal**:
  * `400 Bad Request`: `{"code": "INVALID_PAYMENT_PROOF", "message": "format bukti pembayaran harus PDF, JPG, atau PNG"}`
  * `400 Bad Request`: `{"code": "INVALID_PAYMENT_PROOF", "message": "ukuran bukti pembayaran maksimal 5 MB"}`
  * `404 Not Found`: `{"code": "NOT_FOUND", "message": "purchase order tidak ditemukan"}`
  * `409 Conflict`: `{"code": "CONFLICT", "message": "purchase order sudah dibayar"}`
* **User Story · AC**:
  * **`M11-US3` · AC-1, AC-2, AC-3**: Validasi format berkas bukti transfer, penyimpanan fisik ke cloud storage S3, dan perubahan status pembayaran PO menjadi `paid`.

---

### 6. `GET /purchase-orders/unpaid`
* **Method**: `GET`
* **Autentikasi**: Bearer token (Administrator / Keuangan)
* **Deskripsi**: Menampilkan daftar kewajiban tagihan PO yang barangnya sudah diterima namun pembayarannya belum lunas, beserta akumulasi total nilai rupiah terbuka.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "purchase_orders": [
      {
        "id": "PO-2026-007",
        "supplier": "CV Prima Abadi Furniture",
        "total": 23500000
      }
    ],
    "total": 23500000
  }
  ```
* **Respons Gagal (`500 Internal Server Error`)**:
  ```json
  {
    "error": "database query failed"
  }
  ```
* **User Story · AC**:
  * **`M11-US3` · AC-1**: Pengelolaan daftar tagihan terbuka yang siap dibayar oleh Bagian Keuangan.

---

### 7. `GET /purchase-orders`
* **Method**: `GET`
* **Autentikasi**: Bearer token (Administrator)
* **Query Parameter**: `status` *(string, opsional)*
* **Deskripsi**: Menampilkan master daftar seluruh Purchase Order beserta akumulasi total biaya dan progres kuantitas barang yang diterima per item.
* **Respons Berhasil (`200 OK`)**:
  ```json
  [
    {
      "id": "PO-2026-001",
      "supplier": "PT Graha Sarana Mitra",
      "status": "approved",
      "payment": "unpaid",
      "total": 6500000,
      "items": [
        {
          "id": "ITEM-101",
          "name": "Kertas HVS A4 80gr Sinar Dunia",
          "quantity": 100,
          "unit_cost": 65000,
          "qty_received": 0
        }
      ]
    }
  ]
  ```
* **Respons Gagal (`500 Internal Server Error`)**:
  ```json
  {
    "error": "database connection timeout"
  }
  ```
* **User Story · AC**:
  * **`M11-US1` · AC-1**, **`M11-US2` · AC-1**, **`M11-US3` · AC-1**: Monitoring status komprehensif seluruh siklus pesanan.

---

### 8. `GET /purchase-orders/{id}`
* **Method**: `GET`
* **Autentikasi**: Bearer token (Administrator)
* **Parameter Path**: `id` *(string, wajib)*
* **Deskripsi**: Mengambil detail informasi lengkap satu Purchase Order tertentu.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "id": "PO-2026-001",
    "supplier": "PT Graha Sarana Mitra",
    "status": "approved",
    "payment": "unpaid",
    "total": 6500000,
    "items": [
      {
        "id": "ITEM-101",
        "name": "Kertas HVS A4 80gr Sinar Dunia",
        "quantity": 100,
        "unit_cost": 65000,
        "qty_received": 0
      }
    ]
  }
  ```
* **Respons Gagal (`404 Not Found`)**:
  ```json
  {
    "error": "purchase order not found"
  }
  ```
* **User Story · AC**:
  * **`M11-US1` · AC-1** & **`M11-US2` · AC-1**: Rincian item pesanan spesifik.

---

### 9. `POST /purchase-orders`
* **Method**: `POST`
* **Autentikasi**: Bearer token (Administrator)
* **Header**: `Content-Type: application/json`
* **Deskripsi**: Membuat entri Purchase Order baru di sistem.
* **Request Body**:
  ```json
  {
    "id": "PO-2026-011",
    "supplier": "PT Teknologi Nusantara Raya",
    "status": "approved",
    "items": [
      {
        "id": "ITEM-201",
        "name": "Kabel UTP Cat6 Belden 305m",
        "quantity": 5,
        "unit_cost": 1850000
      }
    ]
  }
  ```
* **Respons Berhasil (`201 Created`)**:
  ```json
  {
    "id": "PO-2026-011",
    "supplier": "PT Teknologi Nusantara Raya",
    "status": "approved",
    "payment": "unpaid",
    "total": 9250000,
    "items": [
      {
        "id": "ITEM-201",
        "name": "Kabel UTP Cat6 Belden 305m",
        "quantity": 5,
        "unit_cost": 1850000,
        "qty_received": 0
      }
    ]
  }
  ```
* **Respons Gagal**:
  * `400 Bad Request`: `{"error": "supplier and items are required"}`
  * `409 Conflict`: `{"error": "purchase order already exists"}`
* **User Story · AC**:
  * **`M11-US1` · AC-1**: Pembuatan master Purchase Order.

---

### 10. `POST /purchase-orders/{id}/receipts`
* **Method**: `POST`
* **Autentikasi**: Bearer token (Administrator)
* **Header**: `Content-Type: application/json`
* **Deskripsi**: Mencatat kuantitas barang diterima langsung ke item PO (jalur integrasi ringkas).
* **Request Body**:
  ```json
  {
    "items": [
      {
        "id": "ITEM-101",
        "quantity": 50
      }
    ]
  }
  ```
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "id": "PO-2026-001",
    "supplier": "PT Graha Sarana Mitra",
    "status": "received partial",
    "payment": "unpaid",
    "total": 6500000,
    "items": [
      {
        "id": "ITEM-101",
        "name": "Kertas HVS A4 80gr Sinar Dunia",
        "quantity": 100,
        "unit_cost": 65000,
        "qty_received": 50
      }
    ]
  }
  ```
* **Respons Gagal**:
  * `400 Bad Request`: `{"error": "invalid items input"}`
  * `404 Not Found`: `{"error": "purchase order not found"}`
  * `409 Conflict`: `{"error": "purchase order must be approved"}`
* **User Story · AC**:
  * **`M11-US1` · AC-1** & **`M11-US2` · AC-1, AC-3**: Penyesuaian kuantitas fisik item.

---

### 11. `GET /reports/procurement`
* **Method**: `GET`
* **Autentikasi**: Bearer token (Administrator / Manajer)
* **Query Parameter**: `from` *(string date, wajib)*, `to` *(string date, wajib)*
* **Deskripsi**: Menyajikan rekapitulasi total nominal belanja pengadaan per vendor rekanan dalam rentang periode tanggal yang dipilih.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "from": "2026-10-01",
    "to": "2026-11-30",
    "suppliers": {
      "CV Mandiri Sukses Perkasa": 9250000,
      "PT Graha Sarana Mitra": 6500000,
      "PT Pasundan Cipta Grafika": 24500000,
      "PT Sinar IT Solusindo": 17400000
    }
  }
  ```
* **Respons Gagal**:
  * `400 Bad Request`: `{"error": "query parameters 'from' and 'to' (YYYY-MM-DD) are required"}`
  * `500 Server Error`: `{"error": "database aggregation error"}`
* **User Story · AC**:
  * **`M11-US1`, `M11-US2`, `M11-US3`**: Rekapitulasi finansial seluruh transaksi belanja barang.

---

### 12. `GET /health`
* **Method**: `GET`
* **Autentikasi**: `-` (Publik)
* **Deskripsi**: Memeriksa status kesiapan runtime serverless dan koneksi aktif ke PostgreSQL Neon Cloud.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "serverless": "running",
      "postgres": "connected (Neon)"
    }
  }
  ```
* **Respons Gagal**: `-`
* **User Story · AC**:
  * **`NFR-01`**: Kesiapan dan keandalan sistem pengadaan.

---

### 13. `GET /`
* **Method**: `GET`
* **Autentikasi**: `-` (Publik)
* **Deskripsi**: Mengembalikan pesan sambutan status runtime Vercel dan daftar rute yang didukung.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "success": true,
    "message": "M11 Procurement API Serverless on Vercel is online",
    "database": "PostgreSQL (Neon)",
    "endpoints": [
      "/health",
      "/purchase-orders",
      "/purchase-orders/approved",
      "/purchase-orders/unpaid",
      "/reports/procurement",
      "/goods-receipts"
    ]
  }
  ```
* **Respons Gagal**: `-`
* **User Story · AC**: Infrastruktur Runtime.

---

### 14. `POST /database/reset-and-seed`
* **Method**: `POST`
* **Autentikasi**: Internal Dev / Admin
* **Deskripsi**: Membersihkan data lama dan menginisialisasi 10 dummy Purchase Order serta Goods Receipt bersih dengan alur pengadaan yang realistis.
* **Respons Berhasil (`200 OK`)**:
  ```json
  {
    "success": true,
    "message": "PostgreSQL (Neon) berhasil dibersihkan dan diisi 10 purchase orders & goods receipts bersih"
  }
  ```
* **Respons Gagal (`500 Internal Server Error`)**:
  ```json
  {
    "success": false,
    "error": "database connection failed"
  }
  ```
* **User Story · AC**: Utilitas Pengujian Pengadaan (Testing Environment).

