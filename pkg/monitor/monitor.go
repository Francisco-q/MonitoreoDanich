package monitor

import (
	"fmt"
	"log"
	"time"

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
