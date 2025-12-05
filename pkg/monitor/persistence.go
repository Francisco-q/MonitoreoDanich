package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Persistence maneja el almacenamiento de datos
type Persistence struct {
	config *SystemConfig
}

// NewPersistence crea un nuevo manejador de persistencia
func NewPersistence(config *SystemConfig) *Persistence {
	return &Persistence{
		config: config,
	}
}

// EnsureDataFolder crea la carpeta de datos si no existe
func (p *Persistence) EnsureDataFolder() error {
	return os.MkdirAll(p.config.DatasetFolder, 0755)
}

// LoadOrCreateDataset carga el dataset existente o crea uno nuevo
func (p *Persistence) LoadOrCreateDataset() TrainingDataset {
	data, err := ioutil.ReadFile(p.config.DatasetFile)
	if err != nil {
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

// LoadLastAssignments carga los últimos assignments guardados
func (p *Persistence) LoadLastAssignments() []Assignment {
	data, err := ioutil.ReadFile(p.config.LastAssignmentsFile)
	if err != nil {
		return []Assignment{}
	}

	var assignments []Assignment
	if err := json.Unmarshal(data, &assignments); err != nil {
		return []Assignment{}
	}

	return assignments
}

// SaveLastAssignments guarda los assignments actuales
func (p *Persistence) SaveLastAssignments(assignments []Assignment) error {
	data, err := json.MarshalIndent(assignments, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(p.config.LastAssignmentsFile, data, 0644)
}

// SaveSnapshot guarda un snapshot individual
func (p *Persistence) SaveSnapshot(snapshot DataSnapshot, filename string) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

// SaveDataset guarda el dataset completo
func (p *Persistence) SaveDataset(dataset TrainingDataset) error {
	data, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return err
	}

	// Guardar dataset completo
	if err := ioutil.WriteFile(p.config.DatasetFile, data, 0644); err != nil {
		return err
	}

	// También guardar snapshot diario
	dailyFile := filepath.Join(p.config.DatasetFolder,
		fmt.Sprintf("snapshots_%s.json", time.Now().Format("20060102")))

	return ioutil.WriteFile(dailyFile, data, 0644)
}

// LogChange registra un cambio en el log
func (p *Persistence) LogChange(change ChangeLog) error {
	var logs []ChangeLog

	if data, err := ioutil.ReadFile(p.config.ChangesLogFile); err == nil {
		json.Unmarshal(data, &logs)
	}

	logs = append(logs, change)

	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(p.config.ChangesLogFile, data, 0644)
}
