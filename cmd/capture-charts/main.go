package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"danich/pkg/scraper"
)

func main() {
	baseURL := "http://192.168.121.2"

	// Crear scraper
	chartScraper := scraper.NewChartScraper(baseURL)

	fmt.Println("=== Capturando datos de gráficos de ambos sorters ===")

	// Obtener datos de ambos sorters
	results, err := chartScraper.ScrapeBothSorters()
	if err != nil {
		log.Fatal("Error al obtener datos:", err)
	}

	// Mostrar resumen en consola
	for _, data := range results {
		fmt.Println("\n" + data.Summary())
		fmt.Println("Detalle por SKU:")
		for sku, percentage := range data.Percentages {
			fmt.Printf("  %s: %.1f%%\n", sku, percentage)
		}
	}

	// Guardar en JSON
	outputFile := "chart_data_captured.json"
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatal("Error al convertir a JSON:", err)
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		log.Fatal("Error al guardar archivo:", err)
	}

	fmt.Printf("\n✓ Datos guardados en: %s\n", outputFile)
}
