package monitor

import (
	"fmt"
	"log"
	"strings"
	"time"

	"danich/pkg/scraper"
)

// SnapshotBuilder construye snapshots del estado del sistema
type SnapshotBuilder struct {
	chartScraper *scraper.ChartScraper
}

// NewSnapshotBuilder crea un nuevo constructor de snapshots
func NewSnapshotBuilder(chartScraper *scraper.ChartScraper) *SnapshotBuilder {
	return &SnapshotBuilder{
		chartScraper: chartScraper,
	}
}

// CreateSnapshot genera un snapshot completo del estado actual
func (sb *SnapshotBuilder) CreateSnapshot(timestamp time.Time, assignments []Assignment) DataSnapshot {
	snapshot := DataSnapshot{
		Timestamp:             timestamp.Format("2006-01-02 15:04:05"),
		DateTime:              timestamp,
		Assignments:           assignments,
		TotalCount:            len(assignments),
		BySorter:              make(map[int]int),
		BySalida:              make(map[int]int),
		CalibreBySorter:       make(map[int]map[string]CalibreDistribution),
		CalibreBySalida:       make(map[int]map[string]CalibreDistribution),
		CalibreBySorterSalida: make(map[string]map[string]CalibreDistribution),
	}

	// Contadores básicos
	for _, a := range assignments {
		snapshot.BySorter[a.SorterID]++
		snapshot.BySalida[a.Salida]++
	}

	// Capturar datos de gráficos si está disponible
	if sb.chartScraper != nil {
		sb.captureChartData(&snapshot, assignments)
	}

	return snapshot
}

// captureChartData captura los porcentajes reales de los gráficos
func (sb *SnapshotBuilder) captureChartData(snapshot *DataSnapshot, assignments []Assignment) {
	chartDataList, err := sb.chartScraper.ScrapeBothSorters()
	if err != nil {
		log.Printf("⚠ Error capturando gráficos: %v", err)
		return
	}

	snapshot.ChartData = make(map[int]*scraper.ChartData)

	// Procesar datos de cada sorter
	for _, chartData := range chartDataList {
		snapshot.ChartData[chartData.SorterID] = chartData
		snapshot.CalibreBySorter[chartData.SorterID] = make(map[string]CalibreDistribution)

		// Guardar porcentajes reales por SKU completo
		for sku, percentage := range chartData.Percentages {
			snapshot.CalibreBySorter[chartData.SorterID][sku] = CalibreDistribution{
				Count:      0, // El count manual no es confiable
				Percentage: percentage,
			}
		}

		// Mapear porcentajes a salidas
		sb.mapPercentagesToOutputs(snapshot, chartData, assignments)
	}

	// Calcular distribución global
	sb.calculateGlobalDistribution(snapshot, chartDataList)

	log.Printf("📊 Gráficos capturados: %d sorters con porcentajes reales", len(chartDataList))
}

// mapPercentagesToOutputs mapea los porcentajes a las salidas físicas
func (sb *SnapshotBuilder) mapPercentagesToOutputs(snapshot *DataSnapshot, chartData *scraper.ChartData, assignments []Assignment) {
	for _, assignment := range assignments {
		if assignment.SorterID != chartData.SorterID {
			continue
		}

		realPercent, exists := chartData.Percentages[assignment.SKU]
		if !exists {
			continue
		}

		// Por salida
		if snapshot.CalibreBySalida[assignment.Salida] == nil {
			snapshot.CalibreBySalida[assignment.Salida] = make(map[string]CalibreDistribution)
		}
		snapshot.CalibreBySalida[assignment.Salida][assignment.SKU] = CalibreDistribution{
			Count:      1,
			Percentage: realPercent,
		}

		// Por sorter+salida
		key := fmt.Sprintf("%d-%d", assignment.SorterID, assignment.Salida)
		if snapshot.CalibreBySorterSalida[key] == nil {
			snapshot.CalibreBySorterSalida[key] = make(map[string]CalibreDistribution)
		}
		snapshot.CalibreBySorterSalida[key][assignment.SKU] = CalibreDistribution{
			Count:      1,
			Percentage: realPercent,
		}
	}
}

// calculateGlobalDistribution calcula el promedio global entre sorters
func (sb *SnapshotBuilder) calculateGlobalDistribution(snapshot *DataSnapshot, chartDataList []*scraper.ChartData) {
	snapshot.CalibrePercent = make(map[string]float64)

	if len(chartDataList) == 0 {
		return
	}

	// Copiar datos del primer sorter
	for sku, percent := range chartDataList[0].Percentages {
		snapshot.CalibrePercent[sku] = percent
	}

	// Si hay segundo sorter, promediar
	if len(chartDataList) > 1 {
		for sku, percent := range chartDataList[1].Percentages {
			if existing, exists := snapshot.CalibrePercent[sku]; exists {
				snapshot.CalibrePercent[sku] = (existing + percent) / 2.0
			} else {
				snapshot.CalibrePercent[sku] = percent
			}
		}
	}
}

// ExtractCalibre extrae el calibre del SKU (formato: CALIBRE-CALIDAD-VARIEDAD-LOTE)
func ExtractCalibre(sku string) string {
	if strings.ToLower(sku) == "descarte" {
		return "Descarte"
	}

	parts := strings.Split(sku, "-")
	if len(parts) > 0 {
		calibre := parts[0]
		// Normalizar nombres de calibres
		switch calibre {
		case "J":
			return "Jumbo"
		case "2J":
			return "Doble_Jumbo"
		case "3J":
			return "Triple_Jumbo"
		case "4J":
			return "Cuadruple_Jumbo"
		case "XL":
			return "Extra_Large"
		default:
			return calibre
		}
	}
	return "Desconocido"
}
