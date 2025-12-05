package monitor

import (
	"fmt"
	"log"
	"strings"
	"time"

	"danich/pkg/advisor"
	"danich/pkg/scraper"
)

// Monitor coordina todo el sistema de monitoreo
type Monitor struct {
	config          *SystemConfig
	fetcher         *Fetcher
	persistence     *Persistence
	changeDetector  *ChangeDetector
	snapshotBuilder *SnapshotBuilder
	exporter        *Exporter
	display         *Display
	chartScraper    *scraper.ChartScraper
	nativeAdvisor   *advisor.Advisor
}

// New crea un nuevo monitor con todas sus dependencias
func New() (*Monitor, error) {
	// Cargar configuración
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error cargando configuración: %w", err)
	}

	// Inicializar componentes
	m := &Monitor{
		config:         config,
		fetcher:        NewFetcher(config.AssignmentsURL),
		persistence:    NewPersistence(config),
		changeDetector: NewChangeDetector(),
		exporter:       NewExporter(config.DatasetFolder),
		display:        NewDisplay(config),
	}

	// Inicializar scraper si está habilitado
	if config.CaptureCharts {
		m.chartScraper = scraper.NewChartScraper(config.BaseURL)
		m.snapshotBuilder = NewSnapshotBuilder(m.chartScraper)
		fmt.Println("✓ Scraper de gráficos inicializado")
	} else {
		m.snapshotBuilder = NewSnapshotBuilder(nil)
	}

	// Inicializar advisor nativo
	advisorConfig := advisor.DefaultConfig()
	m.nativeAdvisor = advisor.NewAdvisor(advisorConfig)
	fmt.Println("✓ Advisor nativo inicializado")

	return m, nil
}

// Run inicia el loop de monitoreo
func (m *Monitor) Run() error {
	m.printHeader()

	// Crear carpeta de datos
	if err := m.persistence.EnsureDataFolder(); err != nil {
		return fmt.Errorf("error creando carpeta de datos: %w", err)
	}

	// Cargar estado inicial
	dataset := m.persistence.LoadOrCreateDataset()
	lastAssignments := m.persistence.LoadLastAssignments()

	checkCount := 0
	startTime := time.Now()

	// Loop principal
	for {
		checkCount++
		if err := m.runCycle(checkCount, &dataset, &lastAssignments, startTime); err != nil {
			log.Printf("❌ Error en ciclo #%d: %v\n", checkCount, err)
		}

		fmt.Printf("\nPróxima verificación en %v...\n", m.config.CheckInterval)
		time.Sleep(m.config.CheckInterval)
	}
}

// runCycle ejecuta un ciclo completo de monitoreo
func (m *Monitor) runCycle(checkCount int, dataset *TrainingDataset, lastAssignments *[]Assignment, startTime time.Time) error {
	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05")

	fmt.Printf("\n[%s] Verificación #%d\n", timestamp, checkCount)

	// 1. Obtener assignments actuales
	currentAssignments, err := m.fetcher.FetchAssignments()
	if err != nil {
		return fmt.Errorf("error obteniendo assignments: %w", err)
	}
	fmt.Printf("✓ Obtenidos %d assignments\n", len(currentAssignments))

	// 2. Crear snapshot
	snapshot := m.snapshotBuilder.CreateSnapshot(now, currentAssignments)

	// 3. Detectar cambios
	hasChanged := m.changeDetector.HasChanges(*lastAssignments, currentAssignments)

	if hasChanged || len(*lastAssignments) == 0 {
		if err := m.handleChanges(timestamp, hasChanged, *lastAssignments, currentAssignments, snapshot); err != nil {
			log.Printf("⚠️  Error manejando cambios: %v\n", err)
		}
		*lastAssignments = currentAssignments
	} else {
		fmt.Println("✓ Sin cambios")
	}

	// 4. Actualizar dataset
	dataset.Snapshots = append(dataset.Snapshots, snapshot)
	dataset.TotalSnapshots = len(dataset.Snapshots)
	dataset.CollectionEnd = now

	// 5. Persistir datos
	if err := m.persistence.SaveDataset(*dataset); err != nil {
		log.Printf("⚠️  Error guardando dataset: %v\n", err)
	}

	// 6. Exportar a CSV
	if len(snapshot.ChartData) > 0 {
		if err := m.exporter.ExportToCSV(snapshot); err != nil {
			log.Printf("⚠️  Error exportando a CSV: %v\n", err)
		} else {
			fmt.Println("✓ Datos exportados a training_data.csv")
		}
	}

	// 7. Mostrar estadísticas
	m.display.ShowStats(snapshot, *dataset, startTime)

	// 8. Analizar balance y generar sugerencias
	if checkCount%10 == 0 && len(snapshot.ChartData) >= 2 { // Cada 5 minutos
		m.generateAdvice(snapshot, checkCount)
	}

	return nil
}

// handleChanges maneja la detección y registro de cambios
func (m *Monitor) handleChanges(timestamp string, hasChanged bool, old, new []Assignment, snapshot DataSnapshot) error {
	if hasChanged {
		fmt.Println("🔔 ¡CAMBIOS DETECTADOS!")

		changes := m.changeDetector.DetectChanges(old, new)
		m.changeDetector.DisplayChanges(changes)

		// Registrar cambios
		if err := m.persistence.LogChange(ChangeLog{
			Timestamp:   timestamp,
			ChangeType:  "update",
			Added:       changes.Added,
			Removed:     changes.Removed,
			Modified:    changes.Modified,
			Description: m.changeDetector.FormatChangeSummary(changes),
		}); err != nil {
			return err
		}
	} else {
		fmt.Println("📊 Primera captura de datos")
	}

	// Guardar snapshot y assignments
	if err := m.persistence.SaveSnapshot(snapshot, m.config.CurrentSnapshotFile); err != nil {
		return err
	}

	return m.persistence.SaveLastAssignments(new)
}

// generateAdvice genera sugerencias usando el advisor nativo
func (m *Monitor) generateAdvice(snapshot DataSnapshot, checkCount int) {
	fmt.Printf("\n🤖 ANÁLISIS DE BALANCE (Verificación #%d)\n", checkCount)
	fmt.Println("═" + strings.Repeat("═", 48))

	// Convertir snapshot a formato del advisor nativo
	state := m.convertToAdvisorState(snapshot)

	// Obtener advice del advisor nativo
	advice, err := m.nativeAdvisor.GetAdvice(state)
	if err != nil {
		fmt.Printf("⚠️ Error obteniendo sugerencia: %v\n", err)
		return
	}

	// Mostrar sugerencia
	m.displayAdvice(advice)
}

// convertToAdvisorState convierte DataSnapshot a advisor.SystemState
func (m *Monitor) convertToAdvisorState(snapshot DataSnapshot) advisor.SystemState {
	state := advisor.SystemState{
		Timestamp: snapshot.DateTime,
		Sorter1:   advisor.SorterData{SKUs: make(map[string]advisor.SKUInfo)},
		Sorter2:   advisor.SorterData{SKUs: make(map[string]advisor.SKUInfo)},
	}

	// Convertir datos del Sorter 1
	if chartData, exists := snapshot.ChartData[1]; exists {
		for sku, percentage := range chartData.Percentages {
			if percentage > 0 {
				lines := m.getLinesForSKU(snapshot.Assignments, 1, sku)
				state.Sorter1.SKUs[sku] = advisor.SKUInfo{
					Percentage: percentage,
					Lines:      lines,
				}
			}
		}
	}

	// Convertir datos del Sorter 2
	if chartData, exists := snapshot.ChartData[2]; exists {
		for sku, percentage := range chartData.Percentages {
			if percentage > 0 {
				lines := m.getLinesForSKU(snapshot.Assignments, 2, sku)
				state.Sorter2.SKUs[sku] = advisor.SKUInfo{
					Percentage: percentage,
					Lines:      lines,
				}
			}
		}
	}

	return state
}

// getLinesForSKU obtiene las líneas asignadas a un SKU en un sorter
func (m *Monitor) getLinesForSKU(assignments []Assignment, sorterID int, sku string) []int {
	var lines []int
	skuUpper := strings.ToUpper(sku)

	for _, assignment := range assignments {
		if assignment.SorterID == sorterID && strings.ToUpper(assignment.SKU) == skuUpper {
			// Evitar duplicados
			found := false
			for _, line := range lines {
				if line == assignment.Salida {
					found = true
					break
				}
			}
			if !found {
				lines = append(lines, assignment.Salida)
			}
		}
	}

	return lines
}

// displayAdvice muestra la sugerencia del advisor de forma visual
func (m *Monitor) displayAdvice(advice *advisor.Advice) {
	switch advice.Accion {
	case "mantener":
		fmt.Printf("✅ %s\n", advice.Razon)

	case "mover":
		fmt.Printf("💡 SUGERENCIA DE OPTIMIZACIÓN\n")
		fmt.Println("═" + strings.Repeat("═", 48))
		fmt.Printf("SKU: %s\n", advice.SKU)
		fmt.Printf("Movimiento: Sorter %d → Sorter %d\n", advice.DeSorter, advice.ASorter)
		fmt.Printf("📋 Razón: %s\n", advice.Razon)
		fmt.Printf("🕐 Timestamp: %s\n", advice.Timestamp)
		fmt.Println("═" + strings.Repeat("═", 48))

	default:
		fmt.Printf("ℹ️  %s\n", advice.Razon)
	}
}

// analyzeBalance analiza el balance y genera sugerencias
// analyzeBalance - función deshabilitada temporalmente
/*
func (m *Monitor) analyzeBalance(snapshot DataSnapshot) *advisor.BalanceAdvice {
	// Construir request para el advisor
	request := advisor.AnalysisRequest{
		Sorter1: make(map[string]advisor.SKUData),
		Sorter2: make(map[string]advisor.SKUData),
	}

	// Extraer datos de cada sorter
	for sorterID, chartData := range snapshot.ChartData {
		// Obtener líneas de cada SKU desde assignments
		// Normalizar SKUs a mayúsculas para matching consistente
		skuLines := make(map[string][]int)
		for _, assignment := range snapshot.Assignments {
			if assignment.SorterID == sorterID {
				normalizedSKU := strings.ToUpper(assignment.SKU)
				skuLines[normalizedSKU] = append(skuLines[normalizedSKU], assignment.Salida)
			}
		}

		log.Printf("📊 Sorter %d: %d SKUs en gráfico, %d assignments con líneas",
			sorterID, len(chartData.Percentages), len(skuLines))

		// Construir mapa de SKUData
		for sku, percentage := range chartData.Percentages {
			normalizedSKU := strings.ToUpper(sku)
			lines := skuLines[normalizedSKU]
			if lines == nil {
				lines = []int{}
				log.Printf("   ⚠️  SKU %s tiene %.1f%% pero sin líneas asignadas", sku, percentage)
			}

			skuData := advisor.SKUData{
				Percentage: percentage,
				Lines:      lines,
			}

			if sorterID == 1 {
				request.Sorter1[sku] = skuData
			} else {
				request.Sorter2[sku] = skuData
			}
		}
	}

	// Llamar al advisor
	advice, err := m.advisorClient.Analyze(request)
	if err != nil {
		log.Printf("⚠️  Error obteniendo sugerencia: %v\n", err)
		return nil
	}

	return advice
}
*/

// printHeader muestra el header del monitor
func (m *Monitor) printHeader() {
	separator := "============================================================"
	fmt.Println("=== Monitor de Asignaciones - Recolección de Datos ===")
	fmt.Printf("URL: %s\n", m.config.AssignmentsURL)
	fmt.Printf("Intervalo de verificación: %v\n", m.config.CheckInterval)
	fmt.Printf("Carpeta de datos: %s\n", m.config.DatasetFolder)
	fmt.Printf("Captura de gráficos: %v\n", m.config.CaptureCharts)
	fmt.Println("Presiona Ctrl+C para detener")
	fmt.Println(separator)
}
