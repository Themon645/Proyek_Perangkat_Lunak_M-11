# FE Handoff — M11 Procurement API

## Base URL

Untuk development lokal:

```text
http://localhost:8080
```

Server backend melakukan bind ke `0.0.0.0`, tetapi URL yang dibuka dari browser adalah `localhost`.

Contoh konfigurasi frontend:

```env
VITE_API_URL=http://localhost:8080
```

### Import ke Bruno

File [bruno-environment.json](../../../bruno-environment.json) dapat di-import melalui menu
**Environments > Import** pada Bruno. Setelah di-import, pilih environment **M11 Local**.

Variabel yang tersedia:

| Variabel | Nilai |
|---|---|
| `baseUrl` | `http://localhost:8080` |
| `purchaseOrderId` | `PO-2026-001` |
| `itemId` | `ITEM-101` |
| `receiptNo` | `SJ-2026-001` |

Untuk membuat collection request, import file [openapi.yaml](../../../openapi.yaml)
melalui menu **Import > OpenAPI**. Gunakan `{{baseUrl}}` sebagai base URL request.

## Format Umum Request

Semua request dengan body wajib memakai header:

```http
Content-Type: application/json
```

Format error:

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

## 1. Membuat Purchase Order

```http
POST /purchase-orders
```

Request:

```json
{
  "id": "PO-2026-001",
  "supplier": "PT Contoh Supplier",
  "status": "approved",
  "items": [
    {
      "id": "ITEM-001",
      "name": "Kertas HVS A4",
      "quantity": 100,
      "unit_cost": 50000
    }
  ]
}
```

Field:

- `id`: identifier bisnis yang dipakai endpoint (contoh `PO-2026-001`).
- `uuid`: identifier unik yang dikembalikan backend; FE tidak perlu mengirimnya saat membuat PO.
- `supplier`: wajib.
- `status`: opsional; default `pending`.
- `items`: wajib dan minimal satu item.
- `quantity`: harus lebih besar dari `0`.
- `unit_cost`: tidak boleh negatif.

Response sukses: `201 Created`.

## 2. Mencatat Goods Receipt / GRN

```http
POST /goods-receipts
```

Request dari form FE:

```json
{
  "purchase_order_id": "PO-2026-001",
  "receipt_no": "SJ-2026-001",
  "received_date": "2026-10-27",
  "receiver": "Petugas Gudang",
  "notes": "Barang diterima dalam kondisi baik",
  "items": [
    {
      "item_id": "ITEM-001",
      "qty_received": 50
    }
  ]
}
```

Field:

- `purchase_order_id`: wajib.
- `receipt_no`: wajib; nomor surat jalan/delivery note.
- `received_date`: wajib dengan format `YYYY-MM-DD`.
- `receiver`: opsional.
- `notes`: opsional.
- `items`: wajib dan minimal satu item.
- `item_id`: wajib dan tidak boleh duplikat dalam satu request.
- `qty_received`: wajib lebih besar dari `0`.
- Jika `qty_received` melebihi sisa kuantitas item PO, response `422` berisi
  `code: "QUANTITY_EXCEEDS_REMAINING"` dan pesan:
  `Jumlah diterima melebihi sisa pesanan barang (sisa: <jumlah>)`.

`purchase_order_id`, `item_id`, dan `id` tetap digunakan sebagai identifier bisnis pada API. Backend melakukan resolve identifier tersebut ke UUID sebelum menyimpan relasi database. FE tidak perlu mengirim field foreign key internal.

Response sukses: `201 Created`.

Saat request berhasil, backend secara atomik:

1. Menyimpan header GRN.
2. Menyimpan detail item GRN.
3. Menghitung akumulasi `qty_received` dari seluruh detail GRN item tersebut.
4. Menyimpan total akumulasi pada item PO dan menghitung sisa:
   `sisa = quantity pesanan - total qty_received`.
5. Mengubah status PO:
   - `received partial` jika masih ada sisa.
   - `received` jika semua item terpenuhi.
6. Mengirim event asynchronous ke Modul M9 jika endpoint M9 dikonfigurasi.

Jika salah satu proses gagal, transaksi database dibatalkan.

## 3. Penerimaan Langsung pada Purchase Order

Endpoint lama yang masih tersedia:

```http
POST /purchase-orders/{purchase_order_id}/receipts
```

Request:

```json
{
  "items": [
    {
      "id": "ITEM-001",
      "quantity": 50
    }
  ]
}
```

Untuk form GRN baru, gunakan `POST /goods-receipts` karena endpoint tersebut menyimpan nomor surat jalan, tanggal, petugas, catatan, dan detail penerimaan.

## 4. Riwayat Penerimaan per Purchase Order

```http
GET /purchase-orders/{purchase_order_id}/receipts
```

Endpoint ini mengembalikan seluruh histori multi-receipt PO secara urut berdasarkan
`received_date`. Response sudah diformat sebagai resource API:

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
        "notes": "Barang diterima dalam kondisi baik",
        "items": [
          {
            "item_id": "ITEM-001",
            "item_name": "Kertas HVS A4",
            "qty_received": 50
          }
        ]
      }
    ]
  }
}
```

Backend menggunakan eager loading untuk mengambil detail item dalam query relasi
yang sama. Jika PO tidak ditemukan, response adalah `404`.

## 5. Relasi Identifier

FE menggunakan identifier bisnis berikut pada endpoint:

| Identifier API | Tabel tujuan | Keterangan |
|---|---|---|
| `purchase_order_id` / PO `id` | `purchase_orders.id` | ID PO yang ditampilkan dan dikirim FE |
| `item_id` | `purchase_order_items.id` | ID item milik PO |
| `receipt_no` | `goods_receipts.receipt_no` | Nomor surat jalan yang mudah dibaca |

Database menggunakan UUID sebagai foreign key internal:

| Kolom foreign key | Referensi |
|---|---|
| `purchase_order_items.purchase_order_uuid` | `purchase_orders.uuid` |
| `goods_receipts.purchase_order_uuid` | `purchase_orders.uuid` |
| `goods_receipt_items.receipt_uuid` | `goods_receipts.uuid` |
| `goods_receipt_items.item_uuid` | `purchase_order_items.uuid` |

Kolom internal tersebut tidak perlu ditambahkan ke payload FE. Contoh payload GRN tetap menggunakan `purchase_order_id` dan `item_id` seperti pada bagian 2.

## 6. Daftar Purchase Order Belum Lunas

```http
GET /purchase-orders/unpaid
```

Response:

```json
{
  "purchase_orders": [
    {
      "id": "PO-2026-001",
      "supplier": "PT Contoh Supplier",
      "total": 5000000
    }
  ],
  "total": 5000000
}
```

## 7. Pembayaran Purchase Order

```http
POST /purchase-orders/{purchase_order_id}/payment
```

Request utama menggunakan `multipart/form-data` dengan field file `proof`:

```text
proof: bukti-transfer.pdf
```

Response sukses: `200 OK`.

Aturan berkas:

- Format yang diterima: PDF, JPG, atau PNG.
- Ukuran maksimal: 5 MB.
- Nama file disimpan menggunakan nama acak yang aman.
- File disimpan di `storage/payment-proofs/`.
- Path file dicatat pada `payment_records.proof_path`.

Contoh:

```ts
const form = new FormData();
form.append("proof", fileInput.files[0]);

await fetch(`${API_URL}/purchase-orders/PO-2026-001/payment`, {
  method: "POST",
  body: form,
});
```

Pembayaran hanya dapat dilakukan jika PO berstatus:

- `received`
- `received partial`

Backend mengubah `payment` PO dari `unpaid` menjadi `paid` (status pelunasan/lunas)
dan membuat satu record pada tabel `payment_records`.

Validasi dan error pembayaran:

- Jika bukti pembayaran bukan PDF, JPG, atau PNG, response `400` memakai
  `code: "INVALID_PAYMENT_PROOF"` dengan pesan
  `format bukti pembayaran harus PDF, JPG, atau PNG`.
- Jika ukuran bukti lebih dari 5 MB, response `400` memakai
  `code: "INVALID_PAYMENT_PROOF"` dengan pesan
  `ukuran bukti pembayaran maksimal 5 MB`.
- Jika PO sudah berstatus `paid`, request berikutnya tidak mengubah data dan
  mengembalikan response `409` dengan pesan `purchase order sudah dibayar`.
- Jika file sekaligus memiliki format dan ukuran yang tidak valid, validasi
  format dilaporkan terlebih dahulu.

## 8. Laporan Procurement

```http
GET /reports/procurement?from=2026-10-01&to=2026-10-31
```

Parameter:

- `from`: wajib, format `YYYY-MM-DD`.
- `to`: wajib, format `YYYY-MM-DD`.

Response:

```json
{
  "from": "2026-10-01",
  "to": "2026-10-31",
  "suppliers": {
    "PT Contoh Supplier": 5000000
  }
}
```

## 9. Mapping Status untuk UI

| Backend status | Label FE yang disarankan |
|---|---|
| `pending` | Diajukan |
| `approved` | Disetujui |
| `received partial` | Diterima Sebagian |
| `received` | Diterima Penuh |
| `paid` | Lunas |
| `rejected` | Ditolak |

Status pembayaran terpisah dari status penerimaan:

```text
payment = unpaid | paid
```

### Payment Record

Setiap transaksi pelunasan dapat dicatat pada tabel `payment_records` dengan
relasi UUID ke `purchase_orders`:

- `purchase_order_uuid`: foreign key internal ke `purchase_orders.uuid`.
- `amount`: nominal pelunasan dalam bilangan bulat rupiah.
- `paid_at`: waktu transaksi pelunasan.
- `proof_path`: path berkas bukti pembayaran.
- `created_at`, `updated_at`, `deleted_at`, dan `uuid`: audit data.

Relasi database menggunakan `purchase_order_uuid`; FE tetap menggunakan ID PO
bisnis pada endpoint.

## 10. Identifier dan Audit Data

Semua tabel utama memiliki:

- `id`: identifier bisnis yang dipakai pada request API.
- `uuid`: identifier unik native PostgreSQL yang dipakai sebagai dasar relasi antar tabel.
- `created_at`: waktu record dibuat.
- `updated_at`: waktu record terakhir diperbarui.
- `deleted_at`: waktu soft delete; kosong berarti record masih aktif.

Backend dapat mengembalikan `uuid` pada response, tetapi FE tidak perlu mengirim `uuid`, `created_at`, `updated_at`, atau `deleted_at` saat membuat data.

Penghapusan bersifat soft delete sehingga data tidak langsung dihapus secara fisik dari database.

## 11. Mapping Error

| HTTP status | Arti |
|---:|---|
| `201` | Data berhasil dibuat |
| `200` | Data berhasil diproses |
| `400` | Payload atau parameter tidak masuk akal |
| `404` | PO tidak ditemukan |
| `409` | Status PO tidak memungkinkan untuk diproses |
| `415` | Content-Type bukan `application/json` |
| `422` | Input tidak lolos validasi |
| `500` | Kesalahan server |

FE sebaiknya menampilkan nilai `error` dari response kepada pengguna.

## 12. Contoh Integrasi Fetch

```ts
const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export async function createGoodsReceipt(payload: unknown) {
  const response = await fetch(`${API_URL}/goods-receipts`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error?.message ?? "Gagal menyimpan penerimaan barang");
  }
  return data;
}
```

## 13. Alur Form GRN

1. FE memilih PO target.
2. FE menampilkan item PO dan jumlah yang sudah diterima.
3. FE menghitung sisa secara visual.
4. FE mengirim hanya jumlah penerimaan saat ini pada `qty_received`.
5. Backend memvalidasi jumlah terhadap data PO terbaru.
6. FE menampilkan status baru dari response backend.
7. FE menampilkan pesan sukses setelah menerima status `201`.

Validasi FE membantu UX, tetapi validasi backend tetap menjadi sumber kebenaran.

## Sinkronisasi Modul M9

FE tidak perlu memanggil Modul M9 secara langsung.

Setelah GRN sukses disimpan, backend mengirim data ke endpoint yang dikonfigurasi melalui:

```env
M9_GRN_ENDPOINT=https://host-m9/api/goods-receipts
```

Backend melakukan retry maksimal tiga kali dan mencatat hasilnya di log. FE cukup menunggu response `201` dari endpoint `/goods-receipts`.

## Menjalankan Backend Lokal

Pastikan PostgreSQL aktif dan `.env` berisi:

```env
APP_HOST=0.0.0.0
APP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password-kamu
DB_NAME=araya-dev
DB_SSLMODE=disable
DB_AUTO_MIGRATE=false
DB_SEED_DUMMY=false
```

Jalankan:

```powershell
go run main.go
```

Backend siap digunakan melalui:

```text
http://localhost:8080
```
