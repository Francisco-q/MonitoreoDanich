package monitor

import (
	"fmt"
	"strings"
	"time"
)

// Display maneja la visualización de estadísticas
type Display struct {
	config *SystemConfig
}

// NewDisplay crea un nuevo display
func NewDisplay(config *SystemConfig) *Display {
	return &Display{
		config: config,
	}
}

// ShowStats muestra las estadísticas del sistema
func (d *Display) ShowStats(snapshot DataSnapshot, dataset TrainingDataset, startTime time.Time) {
	duration := time.Since(startTime)

	fmt.Println("\n" + d.repeat("-", 60))
	fmt.Println("📊 Estadísticas de recolección:")
	fmt.Printf("  • Total snapshots: %d\n", dataset.TotalSnapshots)
	fmt.Printf("  • Tiempo de ejecución: %v\n", duration.Round(time.Second))
	fmt.Printf("  • Assignments actuales: %d\n", snapshot.TotalCount)

	d.showSorterStats(snapshot)
	d.showSalidaStats(snapshot)
	d.showGlobalDistribution(snapshot)
	d.showSorterDistributions(snapshot)
	d.showSalidaDistributions(snapshot)

	fmt.Println(d.repeat("-", 60))
}

// showSorterStats muestra estadísticas por sorter
func (d *Display) showSorterStats(snapshot DataSnapshot) {
	if len(snapshot.BySorter) > 0 {
		fmt.Print("  • Por sorter: ")
		for i := 1; i <= d.config.PackingSorters; i++ {
			if count, exists := snapshot.BySorter[i]; exists {
				fmt.Printf("Sorter %d=%d ", i, count)
			}
		}
		fmt.Println()
	}
}

// showSalidaStats muestra estadísticas por salida
func (d *Display) showSalidaStats(snapshot DataSnapshot) {
	if len(snapshot.BySalida) > 0 {
		fmt.Print("  • Por salida: ")
		for salida := 1; salida <= d.config.PackingLineas; salida++ {
			if count, exists := snapshot.BySalida[salida]; exists {
				fmt.Printf("S%d=%d ", salida, count)
			}
		}
		fmt.Println()
	}
}

// showGlobalDistribution muestra la distribución global
func (d *Display) showGlobalDistribution(snapshot DataSnapshot) {
	if len(snapshot.CalibrePercent) > 0 {
		fmt.Println("\n  • Distribución global (promedio entre sorters):")
		for sku, percent := range snapshot.CalibrePercent {
			fmt.Printf("    - %s: %.0f%%\n", sku, percent)
		}
	}
}

// showSorterDistributions muestra distribuciones por sorter
func (d *Display) showSorterDistributions(snapshot DataSnapshot) {
	if len(snapshot.CalibreBySorter) > 0 && len(snapshot.ChartData) > 0 {
		fmt.Println("\n  • Distribución por Sorter (datos reales del gráfico):")
		for sorterID := 1; sorterID <= d.config.PackingSorters; sorterID++ {
			if chartData, hasChart := snapshot.ChartData[sorterID]; hasChart {
				fmt.Printf("    Sorter %d:\n", sorterID)
				for _, sku := range chartData.OrderedSKUs {
					if dist, exists := snapshot.CalibreBySorter[sorterID][sku]; exists {
						fmt.Printf("      %s: %.0f%%\n", sku, dist.Percentage)
					}
				}
			}
		}
	}
}

// showSalidaDistributions muestra distribuciones por salida
func (d *Display) showSalidaDistributions(snapshot DataSnapshot) {
	if len(snapshot.CalibreBySalida) > 0 {
		fmt.Println("\n  • Distribución por Salida:")
		for salida := 1; salida <= d.config.PackingLineas; salida++ {
			if skus, exists := snapshot.CalibreBySalida[salida]; exists {
				fmt.Printf("    Salida %d:\n", salida)
				for sku, dist := range skus {
					fmt.Printf("      %s: %.0f%%\n", sku, dist.Percentage)
				}
			}
		}
	}
}

// repeat repite un string n veces
func (d *Display) repeat(s string, count int) string {
	return strings.Repeat(s, count)
}
