# M11 - Penerimaan dan Pembayaran Purchase Order

Backend Go untuk pencatatan penerimaan barang, pembayaran PO, daftar PO belum
dibayar, dan rekap pengadaan per supplier.

## Struktur

```text
application/  # service dan aturan bisnis
common/       # konstanta status
controller/   # handler HTTP
database/     # koneksi dan migrasi GORM
dto/          # request DTO
model/        # entity database
repository/   # query dan persistensi GORM
api/          # registrasi API dan route
```

Alur request:

```text
api/routes -> controller -> application -> repository -> database
```

## Menjalankan

```bash
go mod tidy
go run .
```

Server berjalan di `http://localhost:8080`. Database SQLite dibuat sebagai
`m11.db`.

## Pengujian

```bash
go test ./...
```

## API

Semua request `POST` menggunakan header:

```text
Content-Type: application/json
```

### Membuat PO

`POST /purchase-orders`

```json
{
  "id": "PO-001",
  "supplier": "Supplier A",
  "status": "approved",
  "items": [
    {
      "id": "item-001",
      "name": "Laptop",
      "quantity": 50,
      "unit_cost": 10000000
    }
  ]
}
```

Jika `status` tidak dikirim, status awalnya `pending`.

### Mencatat penerimaan

`POST /purchase-orders/{id}/receipts`

```json
{
  "items": [
    {
      "id": "item-001",
      "quantity": 30
    }
  ]
}
```

PO harus berstatus `approved`. Sistem akan menetapkan:

- `received partial` jika jumlah belum lengkap
- `received` jika seluruh jumlah sudah diterima

Jumlah diterima yang melebihi jumlah pesanan ditolak.

### Menandai PO sudah dibayar

`POST /purchase-orders/{id}/payment`

```json
{
  "proof": "transfer-001.jpg"
}
```

PO harus sudah diterima dan bukti transfer wajib diisi.

### Melihat PO belum dibayar

`GET /purchase-orders/unpaid`

Response berisi daftar PO, total tagihan masing-masing, dan total keseluruhan.
Jika tidak ada data, `purchase_orders` bernilai array kosong dan `total`
bernilai `0`.

### Rekap pengadaan

`GET /reports/procurement?from=2026-01-01&to=2026-12-31`

Response berisi total pengadaan per supplier. Jika tidak ada PO pada periode
tersebut, `suppliers` bernilai object kosong.
