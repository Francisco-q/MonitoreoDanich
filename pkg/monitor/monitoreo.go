package monitor

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"danich/pkg/scraper"

	"gopkg.in/yaml.v3"
)

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

type DataSnapshot struct {
	Timestamp   string                     `json:"timestamp"`
	DateTime    time.Time                  `json:"datetime"`
	Assignments []Assignment               `json:"assignments"`
	TotalCount  int                        `json:"total_count"`
	BySorter    map[int]int                `json:"by_sorter"`            // Conteo por sorter
	BySalida    map[int]int                `json:"by_salida"`            // Conteo por salida
	ChartData   map[int]*scraper.ChartData `json:"chart_data,omitempty"` // Datos de gráficos por sorter

	// Distribuciones detalladas
	CalibrePercent        map[string]float64                        `json:"calibre_percent,omitempty"`          // Porcentajes globales
	CalibreBySorter       map[int]map[string]CalibreDistribution    `json:"calibre_by_sorter,omitempty"`        // Por sorter
	CalibreBySalida       map[int]map[string]CalibreDistribution    `json:"calibre_by_salida,omitempty"`        // Por salida
	CalibreBySorterSalida map[string]map[string]CalibreDistribution `json:"calibre_by_sorter_salida,omitempty"` // Por sorter+salida (key: "sorter_id-salida")
}

type TrainingDataset struct {
	CollectionStart time.Time      `json:"collection_start"`
	CollectionEnd   time.Time      `json:"collection_end"`
	TotalSnapshots  int            `json:"total_snapshots"`
	Snapshots       []DataSnapshot `json:"snapshots"`
}

type ChangeLog struct {
	Timestamp   string               `json:"timestamp"`
	ChangeType  string               `json:"change_type"`
	Added       []Assignment         `json:"added,omitempty"`
	Removed     []Assignment         `json:"removed,omitempty"`
	Modified    []ModifiedAssignment `json:"modified,omitempty"`
	Description string               `json:"description"`
}

type ModifiedAssignment struct {
	Old Assignment `json:"old"`
	New Assignment `json:"new"`
}

// Config structs para YAML
type PackingConfig struct {
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	Sorters int    `yaml:"sorters"`
	Lineas  int    `yaml:"lineas"`
	Fruta   string `yaml:"fruta"`
}

type MonitorConfig struct {
	IntervaloSegundos int  `yaml:"intervalo_segundos"`
	CaptureCharts     bool `yaml:"capture_charts"`
}

type DataConfig struct {
	Folder string `yaml:"folder"`
}

type Config struct {
	Packing PackingConfig `yaml:"packing"`
	Monitor MonitorConfig `yaml:"monitor"`
	Data    DataConfig    `yaml:"data"`
}

var (
	baseURL        = "http://192.168.121.2"
	assignmentsURL = baseURL + "/api/api/assignments_list"
	checkInterval  = 30 * time.Second
	captureCharts  = true // Activar captura de gráficos

	// Archivos para el dataset de entrenamiento
	datasetFolder       = "training_data"
	currentSnapshotFile = filepath.Join(datasetFolder, "current_snapshot.json")
	datasetFile         = filepath.Join(datasetFolder, "dataset.json")
	changesLogFile      = filepath.Join(datasetFolder, "changes_log.json")

	// Archivos de estado
	lastAssignmentsFile = "last_assignments.json"

	// Scraper de gráficos
	chartScraper *scraper.ChartScraper
)

// loadConfig carga la configuración desde config.yaml y actualiza las variables globales
func loadConfig() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Printf("⚠️  No se pudo cargar config.yaml, usando valores por defecto: %v\n", err)
		return
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Printf("⚠️  Error parseando config.yaml, usando valores por defecto: %v\n", err)
		return
	}

	// Actualizar variables globales con valores del YAML
	if config.Packing.URL != "" {
		baseURL = config.Packing.URL
		assignmentsURL = baseURL + "/api/api/assignments_list"
	}

	if config.Monitor.IntervaloSegundos > 0 {
		checkInterval = time.Duration(config.Monitor.IntervaloSegundos) * time.Second
	}

	captureCharts = config.Monitor.CaptureCharts

	if config.Data.Folder != "" {
		datasetFolder = config.Data.Folder
		// Actualizar rutas de archivos con el nuevo folder
		currentSnapshotFile = filepath.Join(datasetFolder, "current_snapshot.json")
		datasetFile = filepath.Join(datasetFolder, "dataset.json")
		changesLogFile = filepath.Join(datasetFolder, "changes_log.json")
	}

	fmt.Printf("✓ Configuración cargada: %s (%s) - %d sorters, %d líneas\n", 
		config.Packing.Name, config.Packing.Fruta, config.Packing.Sorters, config.Packing.Lineas)
}

// Run inicia el monitor de asignaciones
func Run() {
	// Cargar configuración desde YAML
	loadConfig()

	fmt.Println("=== Monitor de Asignaciones - Recolección de Datos ===")
	fmt.Printf("URL: %s\n", assignmentsURL)
	fmt.Printf("Intervalo de verificación: %v\n", checkInterval)
	fmt.Printf("Carpeta de datos: %s\n", datasetFolder)
	fmt.Printf("Captura de gráficos: %v\n", captureCharts)
	fmt.Println("Presiona Ctrl+C para detener")
	fmt.Println(repeat("=", 60))
	fmt.Println("=== Monitor de Asignaciones - Recolección de Datos ===")
	fmt.Printf("URL: %s\n", assignmentsURL)
	fmt.Printf("Intervalo de verificación: %v\n", checkInterval)
	fmt.Printf("Carpeta de datos: %s\n", datasetFolder)
	fmt.Printf("Captura de gráficos: %v\n", captureCharts)
	fmt.Println("Presiona Ctrl+C para detener")
	fmt.Println(repeat("=", 60))

	// Crear carpeta de datos si no existe
	if err := os.MkdirAll(datasetFolder, 0755); err != nil {
		log.Fatal("Error creando carpeta de datos:", err)
	}

	// Inicializar scraper de gráficos si está habilitado
	if captureCharts {
		chartScraper = scraper.NewChartScraper(baseURL)
		fmt.Println("✓ Scraper de gráficos inicializado")
	}

	// Cargar o inicializar dataset
	dataset := loadOrCreateDataset()

	// Cargar última versión para detectar cambios
	lastAssignments := loadLastAssignments()

	checkCount := 0
	startTime := time.Now()

	// Loop infinito
	for {
		checkCount++
		now := time.Now()
		timestamp := now.Format("2006-01-02 15:04:05")

		fmt.Printf("\n[%s] Verificación #%d\n", timestamp, checkCount)

		// Obtener assignments actuales
		currentAssignments, err := fetchAssignments()
		if err != nil {
			log.Printf("❌ Error al obtener assignments: %v\n", err)
			time.Sleep(checkInterval)
			continue
		}

		fmt.Printf("✓ Obtenidos %d assignments\n", len(currentAssignments))

		// Crear snapshot con análisis
		snapshot := createSnapshot(now, currentAssignments)

		// Detectar si hay cambios
		hasChanged := hasChanges(lastAssignments, currentAssignments)

		if hasChanged || len(lastAssignments) == 0 {
			if hasChanged {
				fmt.Println("🔔 ¡CAMBIOS DETECTADOS!")

				// Analizar y mostrar cambios
				changes := detectChanges(lastAssignments, currentAssignments)
				displayChanges(changes)

				// Registrar cambios en log
				logChange(ChangeLog{
					Timestamp:   timestamp,
					ChangeType:  "update",
					Added:       changes.Added,
					Removed:     changes.Removed,
					Modified:    changes.Modified,
					Description: formatChangeSummary(changes),
				})
			} else {
				fmt.Println("📊 Primera captura de datos")
			}

			// Guardar snapshot actual
			saveSnapshot(snapshot, currentSnapshotFile)

			// Guardar assignments actuales para próxima comparación
			saveLastAssignments(currentAssignments)

			lastAssignments = currentAssignments
		} else {
			fmt.Println("✓ Sin cambios")
		}

		// Siempre agregar al dataset (para análisis temporal)
		dataset.Snapshots = append(dataset.Snapshots, snapshot)
		dataset.TotalSnapshots = len(dataset.Snapshots)
		dataset.CollectionEnd = now

		// Guardar dataset actualizado
		saveDataset(dataset)

		// Exportar a CSV si hay datos de gráficos
		if len(snapshot.ChartData) > 0 {
			if err := exportToCSV(snapshot); err != nil {
				log.Printf("⚠️  Error exportando a CSV: %v\n", err)
			} else {
				fmt.Println("✓ Datos exportados a training_data.csv")
			}
		}

		// Mostrar estadísticas
		displayStats(snapshot, dataset, startTime)

		// Esperar antes de la siguiente verificación
		fmt.Printf("\nPróxima verificación en %v...\n", checkInterval)
		time.Sleep(checkInterval)
	}
}

func createSnapshot(timestamp time.Time, assignments []Assignment) DataSnapshot {
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

	// Capturar datos de gráficos si está habilitado (DATOS REALES DEL GRÁFICO)
	if captureCharts && chartScraper != nil {
		chartDataList, err := chartScraper.ScrapeBothSorters()
		if err != nil {
			log.Printf("⚠ Error capturando gráficos: %v", err)
		} else {
			snapshot.ChartData = make(map[int]*scraper.ChartData)

			// Usar porcentajes REALES del gráfico, no calculados
			for _, chartData := range chartDataList {
				snapshot.ChartData[chartData.SorterID] = chartData

				// Reemplazar las distribuciones calculadas con los datos REALES del gráfico
				snapshot.CalibreBySorter[chartData.SorterID] = make(map[string]CalibreDistribution)

				// Para cada SKU en el gráfico, usar su NOMBRE COMPLETO y porcentaje REAL
				for sku, percentage := range chartData.Percentages {
					// Usar el SKU completo directamente (ej: "4J-D-SANTINA-C5WFTFG")
					snapshot.CalibreBySorter[chartData.SorterID][sku] = CalibreDistribution{
						Count:      0, // El count manual no es confiable
						Percentage: percentage,
					}
				}

				// Actualizar también las distribuciones por salida usando porcentajes reales
				// Buscar qué assignments de este sorter están en qué salida y asignar porcentajes
				for _, assignment := range assignments {
					if assignment.SorterID == chartData.SorterID {
						// Buscar el porcentaje real de este SKU en el gráfico
						if realPercent, exists := chartData.Percentages[assignment.SKU]; exists {
							// Usar el SKU COMPLETO, no el calibre extraído

							if snapshot.CalibreBySalida[assignment.Salida] == nil {
								snapshot.CalibreBySalida[assignment.Salida] = make(map[string]CalibreDistribution)
							}

							// Guardar con el nombre completo del SKU
							snapshot.CalibreBySalida[assignment.Salida][assignment.SKU] = CalibreDistribution{
								Count:      1,
								Percentage: realPercent,
							}

							// También para sorter+salida con nombre completo
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
				}
			}
			// Calcular distribución GLOBAL usando SKUs completos
			snapshot.CalibrePercent = make(map[string]float64)
			for sku, percent := range chartDataList[0].Percentages {
				snapshot.CalibrePercent[sku] = percent
			}

			// Si hay 2 sorters, promediar por SKU
			if len(chartDataList) > 1 {
				for sku, percent := range chartDataList[1].Percentages {
					if existing, exists := snapshot.CalibrePercent[sku]; exists {
						snapshot.CalibrePercent[sku] = (existing + percent) / 2.0
					} else {
						snapshot.CalibrePercent[sku] = percent
					}
				}
			}

			log.Printf("📊 Gráficos capturados: %d sorters con porcentajes reales", len(chartDataList))
		}
	}

	return snapshot
}

// extractCalibre extrae el calibre del SKU (formato: CALIBRE-CALIDAD-VARIEDAD-LOTE)
func extractCalibre(sku string) string {
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

func fetchAssignments() ([]Assignment, error) {
	resp, err := http.Get(assignmentsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("código de estado: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var assignments []Assignment
	if err := json.Unmarshal(body, &assignments); err != nil {
		return nil, err
	}

	return assignments, nil
}

func loadOrCreateDataset() TrainingDataset {
	data, err := ioutil.ReadFile(datasetFile)
	if err != nil {
		// Crear nuevo dataset
		fmt.Println("✓ Iniciando nuevo dataset")
		return TrainingDataset{
			CollectionStart: time.Now(),
			CollectionEnd:   time.Now(),
			TotalSnapshots:  0,
			Snapshots:       []DataSnapshot{},
		}
	}

	var dataset TrainingDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		log.Printf("⚠ Error al cargar dataset, creando nuevo: %v\n", err)
		return TrainingDataset{
			CollectionStart: time.Now(),
			CollectionEnd:   time.Now(),
			TotalSnapshots:  0,
			Snapshots:       []DataSnapshot{},
		}
	}

	fmt.Printf("✓ Dataset cargado: %d snapshots desde %s\n",
		dataset.TotalSnapshots,
		dataset.CollectionStart.Format("2006-01-02 15:04:05"))

	return dataset
}

func loadLastAssignments() []Assignment {
	data, err := ioutil.ReadFile(lastAssignmentsFile)
	if err != nil {
		return []Assignment{}
	}

	var assignments []Assignment
	if err := json.Unmarshal(data, &assignments); err != nil {
		return []Assignment{}
	}

	return assignments
}

func saveLastAssignments(assignments []Assignment) error {
	data, err := json.MarshalIndent(assignments, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(lastAssignmentsFile, data, 0644)
}

func saveSnapshot(snapshot DataSnapshot, filename string) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func saveDataset(dataset TrainingDataset) error {
	data, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return err
	}

	// Guardar dataset completo
	if err := ioutil.WriteFile(datasetFile, data, 0644); err != nil {
		return err
	}

	// También guardar snapshot diario compacto
	dailyFile := filepath.Join(datasetFolder,
		fmt.Sprintf("snapshots_%s.json", time.Now().Format("20060102")))

	return ioutil.WriteFile(dailyFile, data, 0644)
}

func hasChanges(old, new []Assignment) bool {
	if len(old) != len(new) {
		return true
	}

	oldJSON, _ := json.Marshal(old)
	newJSON, _ := json.Marshal(new)

	return !bytes.Equal(oldJSON, newJSON)
}

type ChangeDetail struct {
	Added    []Assignment
	Removed  []Assignment
	Modified []ModifiedAssignment
}

func detectChanges(old, new []Assignment) ChangeDetail {
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

	for key, oldAssignment := range oldMap {
		if _, exists := newMap[key]; !exists {
			changes.Removed = append(changes.Removed, oldAssignment)
		}
	}

	return changes
}

func displayChanges(changes ChangeDetail) {
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

func displayStats(snapshot DataSnapshot, dataset TrainingDataset, startTime time.Time) {
	duration := time.Since(startTime)

	fmt.Println("\n" + repeat("-", 60))
	fmt.Println("📊 Estadísticas de recolección:")
	fmt.Printf("  • Total snapshots: %d\n", dataset.TotalSnapshots)
	fmt.Printf("  • Tiempo de ejecución: %v\n", duration.Round(time.Second))
	fmt.Printf("  • Assignments actuales: %d\n", snapshot.TotalCount)
	fmt.Printf("  • Por sorter: Sorter 1=%d, Sorter 2=%d\n",
		snapshot.BySorter[1], snapshot.BySorter[2])

	// Mostrar distribución por salida
	fmt.Print("  • Por salida: ")
	for salida := 1; salida <= 7; salida++ {
		if count, exists := snapshot.BySalida[salida]; exists {
			fmt.Printf("S%d=%d ", salida, count)
		}
	}
	fmt.Println()

	// Mostrar distribuciones de SKUs (porcentajes reales del gráfico)
	if len(snapshot.CalibrePercent) > 0 {
		fmt.Println("\n  • Distribución global (promedio entre sorters):")
		for sku, percent := range snapshot.CalibrePercent {
			fmt.Printf("    - %s: %.0f%%\n", sku, percent)
		}
	}

	if len(snapshot.CalibreBySorter) > 0 && len(snapshot.ChartData) > 0 {
		fmt.Println("\n  • Distribución por Sorter (datos reales del gráfico):")
		for sorterID := 1; sorterID <= 2; sorterID++ {
			if chartData, hasChart := snapshot.ChartData[sorterID]; hasChart {
				fmt.Printf("    Sorter %d:\n", sorterID)
				// Usar el orden del gráfico
				for _, sku := range chartData.OrderedSKUs {
					if dist, exists := snapshot.CalibreBySorter[sorterID][sku]; exists {
						fmt.Printf("      %s: %.0f%%\n", sku, dist.Percentage)
					}
				}
			}
		}
	}

	if len(snapshot.CalibreBySalida) > 0 {
		fmt.Println("\n  • Distribución por Salida:")
		for salida := 1; salida <= 7; salida++ {
			if skus, exists := snapshot.CalibreBySalida[salida]; exists {
				fmt.Printf("    Salida %d:\n", salida)
				for sku, dist := range skus {
					fmt.Printf("      %s: %.0f%%\n", sku, dist.Percentage)
				}
			}
		}
	}
	
	if len(snapshot.ChartData) == 2 {
	advice, err := advisor.GetAdvice(snapshot)
	if err == nil && advice.Accion == "mover" {
		fmt.Println("\n🤖:")
		fmt.Printf(" Mueve el %s del Sorter %d al Sorter %d en las líneas %s\n", 
			advice.SKU, advice.DeSorter, advice.ASorter, advice.LineasSugeridas)
		fmt.Printf("   → %s\n", advice.Razon)
		fmt.Printf("   → %s\n", advice.BalanceEsperado)
		fmt.Println()
	}
}

	fmt.Println(repeat("-", 60))
}

func formatChangeSummary(changes ChangeDetail) string {
	return fmt.Sprintf("Agregados: %d, Eliminados: %d, Modificados: %d",
		len(changes.Added), len(changes.Removed), len(changes.Modified))
}

func logChange(change ChangeLog) {
	var logs []ChangeLog

	if data, err := ioutil.ReadFile(changesLogFile); err == nil {
		json.Unmarshal(data, &logs)
	}

	logs = append(logs, change)

	// Mantener todos los cambios (no limitar)
	data, _ := json.MarshalIndent(logs, "", "  ")
	ioutil.WriteFile(changesLogFile, data, 0644)
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

// exportToCSV exporta los datos a formato CSV para entrenamiento de ML
func exportToCSV(snapshot DataSnapshot) error {
	csvFile := filepath.Join(datasetFolder, "training_data.csv")
	
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
			// Parsear SKU para extraer calibre, calidad, variedad
			// Formato típico: "4J-D-SANTINA-C5WFTFG"
			parts := strings.Split(sku, "-")
			calibre := ""
			calidad := ""
			variedad := ""
			
			if len(parts) >= 3 {
				calibre = parts[0]   // "4J"
				calidad = parts[1]   // "D"
				variedad = parts[2]  // "SANTINA"
			}
			
			// Obtener líneas de selladora para este SKU
			lineas := getSalidasForSKU(sorterAssignments, sorterID, sku)
			
			// Crear registro CSV
			record := []string{
				snapshot.Timestamp,
				fmt.Sprintf("%d", sorterID),
				sku,
				calibre,
				calidad,
				variedad,
				lineas,
				fmt.Sprintf("%.1f", percentage),
				fmt.Sprintf("%d", chartData.TotalSKUs),
			}
			
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("error escribiendo registro: %v", err)
			}
		}
	}
	
	return nil
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
