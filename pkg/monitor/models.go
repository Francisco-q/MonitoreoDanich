package monitor

import (
	"time"

	"danich/pkg/scraper"
)

// Assignment representa una asignación de SKU a salida en un sorter
type Assignment struct {
	Salida   int    `json:"salida"`
	SKU      string `json:"sku"`
	SorterID int    `json:"sorter_id"`
}

// CalibreDistribution representa la distribución de calibres con conteo y porcentaje
type CalibreDistribution struct {
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// DataSnapshot representa un snapshot completo del estado del sistema
type DataSnapshot struct {
	Timestamp   string                     `json:"timestamp"`
	DateTime    time.Time                  `json:"datetime"`
	Assignments []Assignment               `json:"assignments"`
	TotalCount  int                        `json:"total_count"`
	BySorter    map[int]int                `json:"by_sorter"`
	BySalida    map[int]int                `json:"by_salida"`
	ChartData   map[int]*scraper.ChartData `json:"chart_data,omitempty"`

	// Distribuciones detalladas
	CalibrePercent        map[string]float64                        `json:"calibre_percent,omitempty"`
	CalibreBySorter       map[int]map[string]CalibreDistribution    `json:"calibre_by_sorter,omitempty"`
	CalibreBySalida       map[int]map[string]CalibreDistribution    `json:"calibre_by_salida,omitempty"`
	CalibreBySorterSalida map[string]map[string]CalibreDistribution `json:"calibre_by_sorter_salida,omitempty"`
}

// TrainingDataset contiene todos los snapshots recolectados
type TrainingDataset struct {
	CollectionStart time.Time      `json:"collection_start"`
	CollectionEnd   time.Time      `json:"collection_end"`
	TotalSnapshots  int            `json:"total_snapshots"`
	Snapshots       []DataSnapshot `json:"snapshots"`
}

// ChangeLog registra un cambio detectado en el sistema
type ChangeLog struct {
	Timestamp   string               `json:"timestamp"`
	ChangeType  string               `json:"change_type"`
	Added       []Assignment         `json:"added,omitempty"`
	Removed     []Assignment         `json:"removed,omitempty"`
	Modified    []ModifiedAssignment `json:"modified,omitempty"`
	Description string               `json:"description"`
}

// ModifiedAssignment representa un assignment que cambió
type ModifiedAssignment struct {
	Old Assignment `json:"old"`
	New Assignment `json:"new"`
}

// ChangeDetail contiene el detalle de los cambios detectados
type ChangeDetail struct {
	Added    []Assignment
	Removed  []Assignment
	Modified []ModifiedAssignment
}
