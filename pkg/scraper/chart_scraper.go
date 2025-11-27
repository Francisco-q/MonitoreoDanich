package scraper

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// ChartData representa los datos del gráfico de porcentajes
type ChartData struct {
	SorterID    int                `json:"sorter_id"`
	Timestamp   time.Time          `json:"timestamp"`
	Percentages map[string]float64 `json:"percentages"`  // SKU -> Porcentaje
	OrderedSKUs []string           `json:"ordered_skus"` // SKUs en el orden del gráfico
	TotalSKUs   int                `json:"total_skus"`
}

// ChartScraper captura datos de porcentajes desde las páginas de assignments
type ChartScraper struct {
	baseURL string
	timeout time.Duration
}

// NewChartScraper crea un nuevo scraper de gráficos
func NewChartScraper(baseURL string) *ChartScraper {
	return &ChartScraper{
		baseURL: baseURL,
		timeout: 30 * time.Second,
	}
}

// ScrapeAssignment obtiene los porcentajes de un assignment específico
func (cs *ChartScraper) ScrapeAssignment(sorterID int) (*ChartData, error) {
	url := fmt.Sprintf("%s/assignment/%d", cs.baseURL, sorterID)

	// Crear contexto con timeout
	ctx, cancel := context.WithTimeout(context.Background(), cs.timeout)
	defer cancel()

	// Crear contexto de Chrome
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var skuData [][]string

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		// Esperar a que cargue el contenido
		chromedp.Sleep(3*time.Second),

		// Extraer todos los divs con la estructura del gráfico
		chromedp.Evaluate(`
			(() => {
				const data = [];
				// Buscar todos los contenedores que tienen SKU + porcentaje
				const containers = document.querySelectorAll('div.relative.w-full.flex.justify-between.items-center');
				
				containers.forEach(container => {
					const h1Elements = container.querySelectorAll('h1.text-xs.font-bold.px-1.text-center');
					if (h1Elements.length >= 2) {
						const sku = h1Elements[0].textContent.trim();
						const percentage = h1Elements[1].textContent.trim();
						data.push([sku, percentage]);
					}
				});
				
				return data;
			})();
		`, &skuData),
	)

	if err != nil {
		return nil, fmt.Errorf("error al hacer scraping: %v", err)
	}

	// Procesar datos manteniendo el orden del gráfico
	chartData := &ChartData{
		SorterID:    sorterID,
		Timestamp:   time.Now(),
		Percentages: make(map[string]float64),
		OrderedSKUs: make([]string, 0),
	}

	for _, item := range skuData {
		if len(item) != 2 {
			continue
		}

		sku := item[0]
		percentageStr := strings.TrimSuffix(item[1], "%")

		percentage, err := strconv.ParseFloat(percentageStr, 64)
		if err != nil {
			log.Printf("Warning: no se pudo parsear porcentaje '%s': %v", item[1], err)
			continue
		}

		chartData.Percentages[sku] = percentage
		chartData.OrderedSKUs = append(chartData.OrderedSKUs, sku) // Mantener orden
		chartData.TotalSKUs++
	}

	return chartData, nil
}

// ScrapeBothSorters obtiene datos de ambos sorters
func (cs *ChartScraper) ScrapeBothSorters() ([]*ChartData, error) {
	results := make([]*ChartData, 0, 2)

	for sorterID := 1; sorterID <= 2; sorterID++ {
		data, err := cs.ScrapeAssignment(sorterID)
		if err != nil {
			log.Printf("Error al obtener datos del sorter %d: %v", sorterID, err)
			continue
		}
		results = append(results, data)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no se pudieron obtener datos de ningún sorter")
	}

	return results, nil
}

// GetCalibreDistribution calcula la distribución de calibres desde los porcentajes
func (cd *ChartData) GetCalibreDistribution() map[string]float64 {
	distribution := make(map[string]float64)

	// Mapear calibres desde SKU
	calibreMap := map[string]string{
		"J":  "Jumbo",
		"2J": "Doble_Jumbo",
		"3J": "Triple_Jumbo",
		"4J": "Cuadruple_Jumbo",
		"XL": "Extra_Large",
	}

	for sku, percentage := range cd.Percentages {
		// Extraer calibre del SKU (formato: CALIBRE-CALIDAD-VARIEDAD-LOTE)
		parts := strings.Split(sku, "-")
		if len(parts) > 0 {
			calibre := parts[0]

			if calibreName, exists := calibreMap[calibre]; exists {
				distribution[calibreName] += percentage
			} else if strings.ToLower(sku) == "descarte" {
				distribution["Descarte"] += percentage
			}
		}
	}

	return distribution
}

// Summary genera un resumen legible de los datos
func (cd *ChartData) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Sorter %d - %s\n", cd.SorterID, cd.Timestamp.Format("15:04:05")))
	sb.WriteString(fmt.Sprintf("Total SKUs: %d\n", cd.TotalSKUs))
	sb.WriteString("Distribución de calibres:\n")

	distribution := cd.GetCalibreDistribution()
	for calibre, percentage := range distribution {
		sb.WriteString(fmt.Sprintf("  %s: %.1f%%\n", calibre, percentage))
	}

	return sb.String()
}
