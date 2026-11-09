# ERD Database M11

Dokumen ini menggambarkan schema database yang dibuat oleh `AutoMigrate` pada
backend saat ini.

## Entity Relationship Diagram

```mermaid
erDiagram
    purchase_orders ||--o{ purchase_order_items : "memiliki"
    purchase_orders ||--o{ goods_receipts : "menerima"
    goods_receipts ||--o{ goods_receipt_items : "memiliki"
    purchase_order_items ||--o{ goods_receipt_items : "diterima pada"
    purchase_orders ||--o{ payment_records : "memiliki"

    purchase_orders {
        string id PK "ID bisnis, contoh PO-2026-001"
        uuid uuid UK "kunci relasi internal"
        string supplier
        string status
        string payment
        string proof
        timestamp received_at
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    purchase_order_items {
        string id PK "ID bisnis item"
        uuid uuid UK "kunci relasi internal"
        uuid purchase_order_uuid FK
        string name
        int quantity
        bigint unit_cost
        int received
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    goods_receipts {
        string receipt_no PK "Nomor surat jalan, contoh SJ-2026/X/9812"
        uuid uuid UK "kunci relasi internal"
        uuid purchase_order_uuid FK
        date received_date
        string receiver
        string notes
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    goods_receipt_items {
        string id PK "ID bisnis detail"
        uuid uuid UK "kunci relasi internal"
        uuid receipt_uuid FK
        uuid item_uuid FK
        int qty_received
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    payment_records {
        uint id PK
        uuid uuid UK "kunci relasi internal"
        uuid purchase_order_uuid FK
        bigint amount
        timestamp paid_at
        string proof_path
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
```

## Relasi

| Tabel sumber | Kolom | Tabel tujuan | Kolom tujuan | Kardinalitas |
|---|---|---|---|---|
| `purchase_order_items` | `purchase_order_uuid` | `purchase_orders` | `uuid` | Banyak item ke satu PO |
| `goods_receipts` | `purchase_order_uuid` | `purchase_orders` | `uuid` | Banyak penerimaan ke satu PO |
| `goods_receipt_items` | `receipt_uuid` | `goods_receipts` | `uuid` | Banyak detail ke satu penerimaan |
| `goods_receipt_items` | `item_uuid` | `purchase_order_items` | `uuid` | Banyak detail ke satu item PO |
| `payment_records` | `purchase_order_uuid` | `purchase_orders` | `uuid` | Banyak pembayaran ke satu PO |

`purchase_order_items` dan `goods_receipts` memiliki relasi melalui
`goods_receipt_items`. Penghapusan PO menghapus item PO dan payment record
terkait secara cascade. Penghapusan goods receipt menghapus detailnya secara
cascade.

## Catatan Identifier

- `id` adalah identifier bisnis yang digunakan pada request API.
- `uuid` adalah identifier internal PostgreSQL untuk foreign key.
- `receipt_no` adalah primary key bisnis pada `goods_receipts`.
- Semua tabel memakai audit field `created_at`, `updated_at`, dan `deleted_at`.
- `deleted_at` dipakai untuk soft delete.
- Nilai uang (`unit_cost` dan `amount`) disimpan sebagai bilangan bulat rupiah.

## Status

### `purchase_orders.status`

```text
pending | approved | received partial | received | rejected
```

### `purchase_orders.payment`

```text
unpaid | paid
```
