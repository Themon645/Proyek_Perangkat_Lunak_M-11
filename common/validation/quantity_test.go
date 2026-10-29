package validation

import "testing"

func TestValidatePartialQuantity(t *testing.T) {
	if err := ValidatePartialQuantity(10, 4, 6); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePartialQuantity(10, 4, 7); err == nil ||
		err.Error() != "Jumlah diterima melebihi sisa pesanan barang (sisa: 6)" {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
