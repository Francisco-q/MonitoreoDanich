package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ChangeDetector maneja la detección de cambios
type ChangeDetector struct{}

// NewChangeDetector crea un nuevo detector de cambios
func NewChangeDetector() *ChangeDetector {
	return &ChangeDetector{}
}

// HasChanges verifica si hay diferencias entre dos listas de assignments
func (cd *ChangeDetector) HasChanges(old, new []Assignment) bool {
	if len(old) != len(new) {
		return true
	}

	oldJSON, _ := json.Marshal(old)
	newJSON, _ := json.Marshal(new)

	return !bytes.Equal(oldJSON, newJSON)
}

// DetectChanges identifica los cambios específicos entre dos estados
func (cd *ChangeDetector) DetectChanges(old, new []Assignment) ChangeDetail {
	changes := ChangeDetail{
		Added:    []Assignment{},
		Removed:  []Assignment{},
		Modified: []ModifiedAssignment{},
	}

	oldMap := make(map[string]Assignment)
	newMap := make(map[string]Assignment)

	for _, a := range old {
		key := fmt.Sprintf("%d-%s", a.SorterID, a.SKU)
		oldMap[key] = a
	}

	for _, a := range new {
		key := fmt.Sprintf("%d-%s", a.SorterID, a.SKU)
		newMap[key] = a
	}

	// Detectar modificados y agregados
	for key, newAssignment := range newMap {
		if oldAssignment, exists := oldMap[key]; exists {
			if oldAssignment.Salida != newAssignment.Salida {
				changes.Modified = append(changes.Modified, ModifiedAssignment{
					Old: oldAssignment,
					New: newAssignment,
				})
			}
		} else {
			changes.Added = append(changes.Added, newAssignment)
		}
	}

	// Detectar eliminados
	for key, oldAssignment := range oldMap {
		if _, exists := newMap[key]; !exists {
			changes.Removed = append(changes.Removed, oldAssignment)
		}
	}

	return changes
}

// FormatChangeSummary crea un resumen textual de los cambios
func (cd *ChangeDetector) FormatChangeSummary(changes ChangeDetail) string {
	return fmt.Sprintf("Agregados: %d, Eliminados: %d, Modificados: %d",
		len(changes.Added), len(changes.Removed), len(changes.Modified))
}

// DisplayChanges muestra los cambios en consola
func (cd *ChangeDetector) DisplayChanges(changes ChangeDetail) {
	if len(changes.Added) > 0 {
		fmt.Printf("\n➕ Agregados (%d):\n", len(changes.Added))
		for _, a := range changes.Added {
			fmt.Printf("   Sorter %d: %s → Salida %d\n", a.SorterID, a.SKU, a.Salida)
		}
	}

	if len(changes.Removed) > 0 {
		fmt.Printf("\n➖ Eliminados (%d):\n", len(changes.Removed))
		for _, a := range changes.Removed {
			fmt.Printf("   Sorter %d: %s (era Salida %d)\n", a.SorterID, a.SKU, a.Salida)
		}
	}

	if len(changes.Modified) > 0 {
		fmt.Printf("\n📝 Modificados (%d):\n", len(changes.Modified))
		for _, m := range changes.Modified {
			fmt.Printf("   Sorter %d: %s - Salida %d → %d\n",
				m.New.SorterID, m.New.SKU, m.Old.Salida, m.New.Salida)
		}
	}
}
