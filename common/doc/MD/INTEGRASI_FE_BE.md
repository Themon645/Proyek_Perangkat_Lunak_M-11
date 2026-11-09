# Panduan Integrasi Frontend ke Backend (FE ke BE)
### Modul 11: Pengadaan, Penerimaan Barang (GRN), & Pelunasan Tagihan
**Araya Business Hub Platform — BPU Universitas Pasundan**

---

## 1. Konfigurasi Lingkungan & Jaringan

* **Base URL**: `http://localhost:8080`
* **Variabel Lingkungan** (`.env`):
  ```env
  VITE_API_URL=http://localhost:8080
  ```
  Backend menerima origin frontend melalui:
  ```env
  CORS_ORIGIN=http://localhost:5173
  ```
* **Header Default**:
  ```http
  Content-Type: application/json
  Accept: application/json
  ```
* **Isolasi Kode**: Semua interaksi HTTP diisolasi di [`src/services/api.js`](file:///Users/tyezarhasan/PPL-FE/src/services/api.js) dan dikelola secara reaktif melalui Pinia Store di [`src/store/po.js`](file:///Users/tyezarhasan/PPL-FE/src/store/po.js). Jika backend offline, frontend beralih otomatis ke *local mock fallback*.

---

## 2. Rute Halaman & Penjelasan Lengkap Tiap Rute

Navigasi diatur secara terpusat di [`src/router/index.js`](file:///Users/tyezarhasan/PPL-FE/src/router/index.js) dengan penerapan *route-level code splitting* (`() => import(...)`):

| Rute URL | Nama Rute | Komponen Halaman | Tiket User Story | Endpoint Backend Terkait |
| :--- | :--- | :--- | :--- | :--- |
| **`/`** | `po-list` | [`POListPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/POListPage.vue) | Master Pengadaan | `GET /purchase-orders`<br>`POST /purchase-orders/:id/payment` |
| **`/grn`** | `grn-form` | [`GRNCreatePage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/GRNCreatePage.vue) | `#SCRUM-285`<br>`#SCRUM-288`<br>`#SCRUM-315` | `POST /goods-receipts`<br>`GET /purchase-orders` |
| **`/tracking`** | `tracking` | [`PartialTrackingPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/PartialTrackingPage.vue) | `#SCRUM-312` | `GET /purchase-orders` |
| **`/pembayaran`** | `payment` | [`PaymentListPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/PaymentListPage.vue) | `#SCRUM-336`<br>`#SCRUM-339` | `GET /purchase-orders/unpaid`<br>`POST /purchase-orders/:id/payment` |
| **`/riwayat-grn`** | `grn-history` | [`GRNHistoryPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/GRNHistoryPage.vue) | Log Arsip GRN | `GET /purchase-orders/:id/receipts` |

---

### 2.1. Rute `/` — Daftar Purchase Order & Tagihan (`po-list`)
* **Komponen**: [`src/pages/POListPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/POListPage.vue)
* **Fungsi Utama**: Halaman utama untuk memonitor siklus seluruh pesanan pengadaan dari status diajukan hingga lunas.
* **Fitur & Logika Klien**:
  * **4 Kartu Metrik Ringkas**: Total PO, Menunggu GRN Masuk, Tagihan Belum Lunas, dan PO Selesai/Lunas.
  * **Filter & Pencarian**: Pencarian instan (nomor PO atau nama vendor) dan dropdown penyaringan status PO.
  * **Paginasi Interaktif**: Navigasi halaman dengan pilihan baris data 5, 10, 25, atau 50 per halaman sesuai Panduan UI Bersama hal 7-8.
  * **Tombol Aksi Kontekstual pada Baris Tabel**:
    * **Catat GRN**: Muncul pada status `Disetujui` atau `Diterima Sebagian`, mengarahkan ke `/grn?po=:id`.
    * **Bayar**: Muncul pada status `Diterima` atau `Diterima Sebagian`, membuka dialog modal pelunasan.
    * **Bukti Bayar**: Muncul pada status `Lunas`, membuka dialog pratinjau berkas bukti transfer.
* **Integrasi Backend**:
  * Saat `onMounted`: Memanggil `poStore.loadFromBackend()` (`GET /purchase-orders`).
  * Saat pelunasan dari modal: Memanggil `POST /purchase-orders/:id/payment`.

---

### 2.2. Rute `/grn` — Form Input Penerimaan Barang (`grn-form`)
* **Komponen**: [`src/pages/GRNCreatePage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/GRNCreatePage.vue)
* **Tiket Terkait**: `#SCRUM-285`, `#SCRUM-288`, `#SCRUM-315`
* **Parameter URL / Query**:
  * `?po=:id` *(Opsional)*: Menentukan nomor PO target yang langsung otomatis terpilih di formulir (misal: `/grn?po=PO-2026-001`).
* **Fungsi Utama**: Formulir pencatatan surat jalan fisik kedatangan barang dari vendor ke gudang BPU.
* **Fitur & Logika Klien**:
  * **Seleksi PO**: Hanya menampilkan PO yang berstatus `Disetujui` atau `Diterima Sebagian`. Jika PO di luar status tersebut dipilih, muncul peringatan penolakan (DoD AC2).
  * **Informasi Surat Jalan**: Input nomor surat jalan/delivery note fisik vendor, tanggal penerimaan, nama petugas gudang (otomatis), dan catatan kondisi barang.
  * **Tabel Verifikasi Item ([`GRNItemTable.vue`](file:///Users/tyezarhasan/PPL-FE/src/components/purchasing/GRNItemTable.vue))**:
    * Menampilkan Kuantitas Dipesan, Diterima Lalu, Input Kuantitas Masuk, Sisa Akhir, dan Bar Progres.
    * Tombol **Maks**: Mengisi otomatis input dengan sisa kuantitas barang yang belum diterima.
    * Validasi Real-time (`#SCRUM-315`): Membatasi kuantitas tidak boleh negatif dan tidak boleh melebihi sisa kekurangan pesanan.
    * Auto-lock & badge "Terpenuhi 100%" pada item yang sudah tuntas diterima.
* **Integrasi Backend**:
  * Mengirim payload dokumen penerimaan ke `POST /goods-receipts` via `poStore.submitGRN()`.
  * Menangani respons `201 Created` dengan mengupdate akumulasi item PO, mencatat riwayat baru, dan mengarahkan kembali ke `/`.
  * Menangani respon error `422 Unprocessable Entity` (`QUANTITY_EXCEEDS_REMAINING`) dengan menampilkan pesan peringatan resmi dari backend.

---

### 2.3. Rute `/tracking` — Tracking Parsial Penerimaan Barang (`tracking`)
* **Komponen**: [`src/pages/PartialTrackingPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/PartialTrackingPage.vue)
* **Tiket Terkait**: `#SCRUM-312`
* **Fungsi Utama**: Visualisasi progres kedatangan fisik barang pesanan yang dikirim secara bertahap (multi-delivery) oleh supplier.
* **Fitur & Logika Klien**:
  * Pencarian cepat berdasarkan nomor PO atau nama supplier.
  * Header kartu PO menampilkan **Total Progress Pemenuhan** seluruh barang pada PO dengan rumus:
    $$\text{Total Progress} = \frac{\sum \text{qty\_diterima\_lalu}}{\sum \text{qty\_order}} \times 100\%$$
  * Tabel rincian per item: Kuantitas dipesan, diterima, sisa kekurangan (penanda hijau jika 0, merah jika masih bersisa), dan bilah progres individual ([`ProgressBar.vue`](file:///Users/tyezarhasan/PPL-FE/src/components/common/ProgressBar.vue)).
* **Integrasi Backend**:
  * Saat `onMounted`: Memanggil `poStore.loadFromBackend()` (`GET /purchase-orders`) untuk memastikan kuantitas item selalu mutakhir.

---

### 2.4. Rute `/pembayaran` — Pelunasan Tagihan PO (`payment`)
* **Komponen**: [`src/pages/PaymentListPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/PaymentListPage.vue)
* **Tiket Terkait**: `#SCRUM-336`, `#SCRUM-339`
* **Fungsi Utama**: Manajemen kewajiban pembayaran tagihan atas barang yang telah masuk ke gudang BPU.
* **Fitur & Logika Klien**:
  * **Banner Total Kewajiban Terbuka**: Akumulasi nilai rupiah seluruh tagihan PO yang barangnya sudah diterima namun belum lunas.
  * **Daftar Tagihan Menunggu Pembayaran**: Tabel rincian PO yang siap dibayar lengkap dengan termin hari dan rekening vendor.
  * **Modal Upload Bukti Transfer ([`PaymentModal.vue`](file:///Users/tyezarhasan/PPL-FE/src/components/purchasing/PaymentModal.vue))**:
    * Area drag-and-drop file resi/bukti pembayaran.
    * Validasi tipe berkas hanya gambar `JPG`, `PNG`, atau file `PDF` dengan batas ukuran maksimal **5 MB**.
    * Pratinjau instan (*image preview* atau icon PDF).
  * **Empty State**: Menampilkan status bersih dan ucapan bahwa seluruh tagihan sudah lunas jika tidak ada kewajiban tertunda (DoD M11-US4 AC2).
* **Integrasi Backend**:
  * Saat `onMounted`: Memanggil `poStore.loadUnpaidPOs()` (`GET /purchase-orders/unpaid`).
  * Saat konfirmasi: Mengirim berkas bukti fisik ke `POST /purchase-orders/:id/payment` via `multipart/form-data` dengan field `proof`.
  * Menangani respon error `400` (`INVALID_PAYMENT_PROOF`) dan `409` (`purchase order sudah dibayar`).
  * Saat sukses (`200 OK`), status PO langsung menjadi `Lunas` (`payment: 'paid'`), file bukti tersimpan, dan baris tagihan otomatis hilang dari daftar kewajiban terbuka.

---

### 2.5. Rute `/riwayat-grn` — Riwayat Dokumen GRN (`grn-history`)
* **Komponen**: [`src/pages/GRNHistoryPage.vue`](file:///Users/tyezarhasan/PPL-FE/src/pages/GRNHistoryPage.vue)
* **Fungsi Utama**: Arsip digital seluruh bukti tanda terima fisik barang masuk (surat jalan / delivery note) dari supplier ke gudang BPU.
* **Fitur & Logika Klien**:
  * Daftar kartu dokumen GRN urut dari yang terbaru.
  * Rincian dokumen: Nomor GRN, Nomor Referensi PO, Tanggal Penerimaan, Nomor Surat Jalan vendor, Petugas Penerima Gudang, dan Catatan Kondisi Barang.
  * Rincian barang yang masuk khusus pada transaksi pengiriman tersebut dengan badge kuantitas (misal: `+40 unit`).
  * Empty state jika belum ada arsip dokumen GRN yang tercatat.
* **Integrasi Backend**:
  * Saat `onMounted`: Memanggil `poStore.loadGRNHistory()` yang mengonsumsi `GET /purchase-orders/:id/receipts`.
  * Menampilkan nama item langsung dari atribut `item_name` respon API atau fallback ke data master PO.

---

## 3. Spesifikasi Kontrak Endpoint API

### 3.1. Daftar Purchase Order
* **Endpoint**: `GET /purchase-orders`
* **Status**: `200 OK`
* **Response**:
  ```json
  [
    {
      "id": "PO-2026-001",
      "supplier": "PT Graha Sarana Mitra",
      "status": "approved",
      "payment": "unpaid",
      "total": 14500000,
      "items": [
        {
          "id": "ITEM-101",
          "name": "Kertas HVS A4",
          "quantity": 100,
          "unit_cost": 55000,
          "qty_received": 40
        }
      ]
    }
  ]
  ```

### 3.2. Detail Purchase Order
* **Endpoint**: `GET /purchase-orders/:id`
* **Status**: `200 OK`
* **Response**: Objek tunggal Purchase Order.

### 3.3. Pembuatan Purchase Order
* **Endpoint**: `POST /purchase-orders`
* **Status**: `201 Created`
* **Request**:
  ```json
  {
    "id": "PO-2026-001",
    "supplier": "PT Graha Sarana Mitra",
    "status": "approved",
    "items": [
      {
        "id": "ITEM-101",
        "name": "Kertas HVS A4",
        "quantity": 100,
        "unit_cost": 55000
      }
    ]
  }
  ```

### 3.4. Pencatatan Penerimaan Barang (GRN)
* **Endpoint**: `POST /goods-receipts`
* **Status**: `201 Created` (Sukses) / `422 Unprocessable Entity` (Jika qty melebihi sisa)
* **Request**:
  ```json
  {
    "purchase_order_id": "PO-2026-001",
    "receipt_no": "SJ-2026-001",
    "received_date": "2026-10-27",
    "receiver": "Tyezar Hasan (Staff Gudang)",
    "notes": "Kondisi baik",
    "items": [
      {
        "item_id": "ITEM-101",
        "qty_received": 50
      }
    ]
  }
  ```
* **Aturan Backend**:
  * Mengakumulasi total `qty_received` per item.
  * Status PO otomatis berubah ke `received partial` (jika masih bersisa) atau `received` (jika 100% tuntas).

### 3.5. Riwayat Penerimaan per PO
* **Endpoint**: `GET /purchase-orders/:id/receipts`
* **Status**: `200 OK`
* **Response**:
  ```json
  {
    "success": true,
    "data": {
      "purchase_order_id": "PO-2026-001",
      "receipts": [
        {
          "receipt_no": "SJ-2026-001",
          "received_date": "2026-10-27",
          "receiver": "Petugas Gudang",
          "notes": "Barang lengkap",
          "items": [
            {
              "item_id": "ITEM-101",
              "item_name": "Kertas HVS A4",
              "qty_received": 50
            }
          ]
        }
      ]
    }
  }
  ```

### 3.6. Daftar Tagihan Belum Lunas
* **Endpoint**: `GET /purchase-orders/unpaid`
* **Status**: `200 OK`
* **Response**:
  ```json
  {
    "purchase_orders": [
      {
        "id": "PO-2026-001",
        "supplier": "PT Graha Sarana Mitra",
        "total": 14500000
      }
    ],
    "total": 14500000
  }
  ```

### 3.7. Pelunasan & Upload Bukti Pembayaran
* **Endpoint**: `POST /purchase-orders/:id/payment`
* **Content-Type**: `multipart/form-data`
* **Status**: `200 OK` (Sukses) / `400 Bad Request` (Format/ukuran salah) / `409 Conflict` (Sudah lunas)
* **Form Fields**:
  * `proof` *(File, Wajib)*: File bukti transfer (`PDF`, `JPG`, `PNG`), maksimal **5 MB**.
  * `bank` *(String, Opsional)*: Nama bank tujuan.
  * `paid_at` *(String YYYY-MM-DD, Opsional)*: Tanggal pembayaran.
* **Aturan Backend**: Hanya dapat dieksekusi jika status PO adalah `received` atau `received partial`. Mengubah field `payment` menjadi `paid`.

### 3.8. Laporan Procurement
* **Endpoint**: `GET /reports/procurement?from=YYYY-MM-DD&to=YYYY-MM-DD`
* **Status**: `200 OK`
* **Response**:
  ```json
  {
    "from": "2026-10-01",
    "to": "2026-10-31",
    "suppliers": {
      "PT Graha Sarana Mitra": 14500000
    }
  }
  ```

---

## 4. Normalisasi Data & Pemetaan Status

### 4.1. Status PO & Badge Antarmuka
| Backend `status` | Backend `payment` | Label Status UI | Warna Badge | Keterangan |
| :--- | :--- | :--- | :--- | :--- |
| `pending` | `unpaid` | **Diajukan** | Kuning (`pending`) | Baru dibuat, belum diverifikasi |
| `approved` | `unpaid` | **Disetujui** | Hijau (`success`) | Siap menerima GRN fisik |
| `received partial` | `unpaid` | **Diterima Sebagian** | Kuning (`pending`) | Masih ada sisa kekurangan |
| `received` | `unpaid` | **Diterima Penuh** | Biru (`info`) | 100% kuantitas telah masuk |
| *(apapun)* | `paid` | **Lunas** | Hijau (`success`) | Tagihan telah dibayar lunas |
| `rejected` | `unpaid` | **Ditolak** | Merah (`danger`) | Pengadaan dibatalkan/ditolak |

### 4.2. Translasi Field Backend $\leftrightarrow$ Frontend
| Data Domain | Kolom Backend API | Properti State Frontend |
| :--- | :--- | :--- |
| **PO Header** | `id` / `nomor_po` | `po.id` & `po.nomor_po` |
| **PO Header** | `supplier` | `po.supplier_name` |
| **PO Header** | `total` | `po.total_tagihan` |
| **PO Item** | `name` | `item.nama_barang` |
| **PO Item** | `quantity` | `item.qty_order` |
| **PO Item** | `unit_cost` | `item.harga_satuan` |
| **PO Item** | `qty_received` / `received_quantity` | `item.qty_diterima_lalu` |
| **GRN Header** | `receipt_no` | `grn.nomor_surat_jalan` & `grn.nomor_grn` |
| **GRN Header** | `received_date` | `grn.tanggal_penerimaan` |
| **GRN Header** | `receiver` | `grn.penerima` |

---

## 5. Standar Format Error Backend

Format respon galat backend:
```json
{
  "success": false,
  "error": {
    "code": "QUANTITY_EXCEEDS_REMAINING",
    "message": "Jumlah diterima melebihi sisa pesanan barang (sisa: 6)",
    "fields": {
      "items": "Jumlah diterima melebihi sisa pesanan barang (sisa: 6)"
    }
  }
}
```

| Kode HTTP | Error Code | Deskripsi & Tindakan FE |
| :---: | :--- | :--- |
| **400** | `INVALID_PAYMENT_PROOF` | Format bukan PDF/JPG/PNG atau ukuran > 5 MB $\rightarrow$ Tampilkan notifikasi danger. |
| **404** | `NOT_FOUND` | Data PO tidak ditemukan $\rightarrow$ Tampilkan empty state / pesan peringatan. |
| **409** | `CONFLICT` | PO sudah berstatus lunas / tidak dapat diproses $\rightarrow$ Kunci tombol pembayaran. |
| **422** | `QUANTITY_EXCEEDS_REMAINING` | Input melebihi sisa pesanan $\rightarrow$ Kunci tombol submit & beri peringatan per baris item. |

---

## 6. Pengujian Singkat Integrasi

1. **Jalankan Backend**:
   ```bash
   go run main.go
   ```
2. **Jalankan Frontend**:
   ```bash
   npm run dev
   ```
3. Akses `http://localhost:5173`. Periksa Network tab di browser:
   * Pemuatan data tabel memanggil `GET /purchase-orders`.
   * Form GRN mengirim `POST /goods-receipts`.
   * Pelunasan mengirim `POST /purchase-orders/:id/payment`.
