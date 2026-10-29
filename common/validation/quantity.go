package validation

import "fmt"

func ValidatePartialQuantity(ordered, received, requested int) error {
	remaining := ordered - received
	if requested > remaining {
		return fmt.Errorf("Jumlah diterima melebihi sisa pesanan barang (sisa: %d)", remaining)
	}
	return nil
}
