package monitor

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Exporter maneja la exportación de datos a CSV
type Exporter struct {
	datasetFolder string
}

// NewExporter crea un nuevo exportador
func NewExporter(datasetFolder string) *Exporter {
	return &Exporter{
		datasetFolder: datasetFolder,
	}
}

// ExportToCSV exporta los datos a formato CSV para entrenamiento de ML
func (e *Exporter) ExportToCSV(snapshot DataSnapshot) error {
	csvFile := filepath.Join(e.datasetFolder, "training_data.csv")

	// Verificar si el archivo existe para decidir si escribir headers
	fileExists := false
	if _, err := os.Stat(csvFile); err == nil {
		fileExists = true
	}

	// Abrir archivo en modo append
	file, err := os.OpenFile(csvFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo archivo CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = ';' // Usar punto y coma para compatibilidad con Excel en español
	defer writer.Flush()

	// Escribir headers solo si el archivo es nuevo
	if !fileExists {
		headers := []string{
			"timestamp",
			"sorter_id",
			"sku",
			"calibre",
			"calidad",
			"variedad",
			"lineas",
			"porcentaje",
			"total_skus_activos",
		}
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("error escribiendo headers: %v", err)
		}
	}

	// Procesar cada sorter con datos de gráfico
	for sorterID, chartData := range snapshot.ChartData {
		if chartData == nil {
			continue
		}

		// Obtener assignments de este sorter
		var sorterAssignments []Assignment
		for _, a := range snapshot.Assignments {
			if a.SorterID == sorterID {
				sorterAssignments = append(sorterAssignments, a)
			}
		}

		// Para cada SKU con porcentaje
		for sku, percentage := range chartData.Percentages {
			record := e.createCSVRecord(snapshot, sorterID, sku, percentage, chartData.TotalSKUs, sorterAssignments)
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("error escribiendo registro: %v", err)
			}
		}
	}

	return nil
}

// createCSVRecord crea un registro CSV para un SKU
func (e *Exporter) createCSVRecord(snapshot DataSnapshot, sorterID int, sku string, percentage float64, totalSKUs int, assignments []Assignment) []string {
	// Parsear SKU para extraer calibre, calidad, variedad
	// Formato típico: "4J-D-SANTINA-C5WFTFG"
	parts := strings.Split(sku, "-")
	calibre := ""
	calidad := ""
	variedad := ""

	if len(parts) >= 3 {
		calibre = parts[0]  // "4J"
		calidad = parts[1]  // "D"
		variedad = parts[2] // "SANTINA"
	}

	// Obtener líneas de selladora para este SKU
	lineas := getSalidasForSKU(assignments, sorterID, sku)

	return []string{
		snapshot.Timestamp,
		fmt.Sprintf("%d", sorterID),
		sku,
		calibre,
		calidad,
		variedad,
		lineas,
		fmt.Sprintf("%.1f", percentage),
		fmt.Sprintf("%d", totalSKUs),
	}
}

// getSalidasForSKU obtiene las líneas (salidas) asignadas a un SKU en formato "L1 L2 L3"
func getSalidasForSKU(assignments []Assignment, sorterID int, sku string) string {
	var salidas []int

	// Normalizar SKU para comparación (convertir a mayúsculas)
	skuNorm := strings.ToUpper(sku)

	for _, a := range assignments {
		if a.SorterID == sorterID && strings.ToUpper(a.SKU) == skuNorm {
			salidas = append(salidas, a.Salida)
		}
	}

	// Ordenar salidas
	sort.Ints(salidas)

	// Convertir a formato "L1 L2 L3"
	var parts []string
	for _, s := range salidas {
		parts = append(parts, fmt.Sprintf("L%d", s))
	}

	return strings.Join(parts, " ")
}
