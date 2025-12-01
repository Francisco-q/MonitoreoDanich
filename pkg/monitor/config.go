package monitor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

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

// SystemConfig contiene toda la configuración del sistema
type SystemConfig struct {
	BaseURL             string
	AssignmentsURL      string
	CheckInterval       time.Duration
	CaptureCharts       bool
	DatasetFolder       string
	CurrentSnapshotFile string
	DatasetFile         string
	ChangesLogFile      string
	LastAssignmentsFile string
	TrainingDataCSV     string
	
	// Info del packing
	PackingName    string
	PackingSorters int
	PackingLineas  int
	PackingFruta   string
}

// LoadConfig carga la configuración desde config.yaml
func LoadConfig() (*SystemConfig, error) {
	// Valores por defecto
	cfg := &SystemConfig{
		BaseURL:             "http://192.168.121.2",
		CheckInterval:       30 * time.Second,
		CaptureCharts:       true,
		DatasetFolder:       "training_data",
		LastAssignmentsFile: "last_assignments.json",
	}

	// Intentar cargar config.yaml
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Printf("⚠️  No se pudo cargar config.yaml, usando valores por defecto: %v\n", err)
		cfg.initDerivedPaths()
		return cfg, nil
	}

	var yamlConfig Config
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		log.Printf("⚠️  Error parseando config.yaml, usando valores por defecto: %v\n", err)
		cfg.initDerivedPaths()
		return cfg, nil
	}

	// Aplicar valores del YAML
	if yamlConfig.Packing.URL != "" {
		cfg.BaseURL = yamlConfig.Packing.URL
	}
	
	if yamlConfig.Monitor.IntervaloSegundos > 0 {
		cfg.CheckInterval = time.Duration(yamlConfig.Monitor.IntervaloSegundos) * time.Second
	}
	
	cfg.CaptureCharts = yamlConfig.Monitor.CaptureCharts
	
	if yamlConfig.Data.Folder != "" {
		cfg.DatasetFolder = yamlConfig.Data.Folder
	}
	
	// Info del packing
	cfg.PackingName = yamlConfig.Packing.Name
	cfg.PackingSorters = yamlConfig.Packing.Sorters
	cfg.PackingLineas = yamlConfig.Packing.Lineas
	cfg.PackingFruta = yamlConfig.Packing.Fruta

	// Inicializar rutas derivadas
	cfg.initDerivedPaths()

	fmt.Printf("✓ Configuración cargada: %s (%s) - %d sorters, %d líneas\n",
		cfg.PackingName, cfg.PackingFruta, cfg.PackingSorters, cfg.PackingLineas)

	return cfg, nil
}

// initDerivedPaths inicializa las rutas que dependen de otras configuraciones
func (cfg *SystemConfig) initDerivedPaths() {
	cfg.AssignmentsURL = cfg.BaseURL + "/api/api/assignments_list"
	cfg.CurrentSnapshotFile = filepath.Join(cfg.DatasetFolder, "current_snapshot.json")
	cfg.DatasetFile = filepath.Join(cfg.DatasetFolder, "dataset.json")
	cfg.ChangesLogFile = filepath.Join(cfg.DatasetFolder, "changes_log.json")
	cfg.TrainingDataCSV = filepath.Join(cfg.DatasetFolder, "training_data.csv")
}
