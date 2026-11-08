// Package identifier contiene validaciones compartidas de identificadores.
package identifier

import "github.com/google/uuid"

// IsCanonicalUUID comprueba el formato hexadecimal con guiones, sin restringir
// la versión o variante. Acepta letras mayúsculas y minúsculas.
func IsCanonicalUUID(value string) bool {
	return len(value) == 36 && uuid.Validate(value) == nil
}

// IsOrderUUID conserva las versiones y la variante admitidas por la API.
func IsOrderUUID(value string) bool {
	if !IsCanonicalUUID(value) {
		return false
	}
	id, err := uuid.Parse(value)
	return err == nil && id.Version() >= 1 && id.Version() <= 5 && id.Variant() == uuid.RFC4122
}
