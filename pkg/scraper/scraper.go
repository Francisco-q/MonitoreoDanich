package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Assignment struct {
	Salida   int    `json:"salida"`
	SKU      string `json:"sku"`
	SorterID int    `json:"sorter_id"`
}

type DataSnapshot struct {
	Timestamp   string       `json:"timestamp"`
	DateTime    time.Time    `json:"datetime"`
	Assignments []Assignment `json:"assignments"`
	TotalCount  int          `json:"total_count"`
	BySorter    map[int]int  `json:"by_sorter"` // Conteo por sorter
	BySalida    map[int]int  `json:"by_salida"` // Conteo por salida
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

var (
	baseURL        = "http://192.168.121.2"
	assignmentsURL = baseURL + "/api/api/assignments_list"
	checkInterval  = 30 * time.Second

	// Archivos para el dataset de entrenamiento
	datasetFolder       = "training_data"
	currentSnapshotFile = filepath.Join(datasetFolder, "current_snapshot.json")
	datasetFile         = filepath.Join(datasetFolder, "dataset.json")
	changesLogFile      = filepath.Join(datasetFolder, "changes_log.json")

	// Archivos de estado
	lastAssignmentsFile = "last_assignments.json"
)

// RunScraper ejecuta el scraper de datos
func RunScraper() {
	fmt.Println("=== Monitor de Asignaciones - Recolección de Datos ===")
	fmt.Printf("URL: %s\n", assignmentsURL)
	fmt.Printf("Intervalo de verificación: %v\n", checkInterval)
	fmt.Printf("Carpeta de datos: %s\n", datasetFolder)
	fmt.Println("Presiona Ctrl+C para detener")
	fmt.Println(repeat("=", 60))

	// Crear carpeta de datos si no existe
	if err := os.MkdirAll(datasetFolder, 0755); err != nil {
		log.Fatal("Error creando carpeta de datos:", err)
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

		// Mostrar estadísticas
		displayStats(snapshot, dataset, startTime)

		// Esperar antes de la siguiente verificación
		fmt.Printf("\nPróxima verificación en %v...\n", checkInterval)
		time.Sleep(checkInterval)
	}
}

func createSnapshot(timestamp time.Time, assignments []Assignment) DataSnapshot {
	snapshot := DataSnapshot{
		Timestamp:   timestamp.Format("2006-01-02 15:04:05"),
		DateTime:    timestamp,
		Assignments: assignments,
		TotalCount:  len(assignments),
		BySorter:    make(map[int]int),
		BySalida:    make(map[int]int),
	}

	// Análisis de distribución
	for _, a := range assignments {
		snapshot.BySorter[a.SorterID]++
		snapshot.BySalida[a.Salida]++
	}

	return snapshot
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

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
