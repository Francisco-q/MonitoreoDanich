# Documentación Técnica - Sistema de Monitoreo de Calibres

**Proyecto:** MonitoreoDanich  
**Propósito:** Recolección automatizada de datos de sorters para entrenamiento de modelos ML  
**Fecha:** Noviembre 2025  
**Lenguaje:** Go 1.25.4

---

## 📑 Tabla de Contenidos

1. [Arquitectura General](#arquitectura-general)
2. [Stack Tecnológico](#stack-tecnológico)
3. [Estructura de Paquetes](#estructura-de-paquetes)
4. [Flujo de Datos](#flujo-de-datos)
5. [Componentes Principales](#componentes-principales)
6. [Modelos de Datos](#modelos-de-datos)
7. [Algoritmos y Lógica](#algoritmos-y-lógica)
8. [Configuración](#configuración)
9. [Dependencias Externas](#dependencias-externas)
10. [Casos de Uso](#casos-de-uso)

---

## 🏗️ Arquitectura General

### Patrón de Diseño
**Hexagonal Architecture + Event-Driven Design**

```
┌────────────────────────────────────────────────────────┐
│                    APLICACIÓN (cmd/)                   │
│  ┌──────────────────┐      ┌────────────────────┐      │
│  │  monitor/        │      │  capture-charts/   │      │
│  │  main.go         │      │  main.go           │      │
│  └──────────────────┘      └────────────────────┘      │
└───────────────────┬──────────────────┬─────────────────┘
                    │                  │
                    ▼                  ▼
┌────────────────────────────────────────────────────────┐
│                   LÓGICA DE NEGOCIO (pkg/)             │
│  ┌──────────────────────────────────────────────┐      │
│  │  monitor/monitoreo.go                        │      │
│  │  - Run()           (loop principal)          │      │
│  │  - createSnapshot() (análisis de datos)      │      │
│  │  - detectChanges() (comparación estados)     │      │
│  └──────────────────────────────────────────────┘      │
│  ┌──────────────────────────────────────────────┐      │
│  │  scraper/                                    │      │
│  │  - chart_scraper.go (chromedp)               │      │
│  └──────────────────────────────────────────────┘      │
└───────────────────┬──────────────────┬─────────────────┘
                    │                  │
                    ▼                  ▼
┌────────────────────────────────────────────────────────┐
│                  INFRAESTRUCTURA                       │
│  ┌──────────────┐  ┌───────────────┐  ┌──────────────┐ │
│  │  HTTP API    │  │  Chromedp     │  │  Filesystem  │ │
│  │  192.168.*   │  │  (Browser)    │  │  JSON files  │ │
│  └──────────────┘  └───────────────┘  └──────────────┘ │
└────────────────────────────────────────────────────────┘
```

### Principios Aplicados

1. **Separation of Concerns**: Cada paquete tiene una responsabilidad clara
2. **Dependency Injection**: ChartScraper se inyecta en el monitor
3. **Single Responsibility**: Funciones pequeñas con un solo propósito
4. **Data Persistence**: Estado guardado en JSON para análisis posterior

---

## 💻 Stack Tecnológico

### Lenguaje Principal
```go
Go 1.25.4
```

**¿Por qué Go?**
- ✅ Concurrencia nativa (goroutines)
- ✅ Binarios compilados sin dependencias
- ✅ Alto rendimiento para scraping
- ✅ Excelente para herramientas CLI

### Librerías Core

#### 1. **chromedp** v0.14.2
```go
import "github.com/chromedp/chromedp"
```
**Propósito:** Controlar Chrome headless para scraping de JavaScript  
**Uso en el proyecto:**
- Navegar a páginas HTML dinámicas
- Ejecutar JavaScript en el DOM
- Extraer datos renderizados (porcentajes de gráficos)

**Ejemplo:**
```go
chromedp.Run(ctx,
    chromedp.Navigate(url),
    chromedp.Sleep(3*time.Second),
    chromedp.Evaluate(`/* JavaScript aquí */`, &resultado),
)
```

#### 2. **goquery** v1.11.0
```go
import "github.com/PuerkitoBio/goquery"
```
**Propósito:** Parsear y consultar documentos HTML (estilo jQuery)  
**Uso:** Análisis de estructura HTML estático

#### 3. **Librería Estándar de Go**
```go
"encoding/json"    // Serialización JSON
"net/http"         // Cliente HTTP
"time"             // Manejo de tiempo
"io/ioutil"        // I/O de archivos
"context"          // Manejo de timeouts
"sort"             // Ordenamiento de slices
```

---

## 📦 Estructura de Paquetes

```
MonitoreoDanich/
│
├── cmd/                           # Puntos de entrada de aplicaciones
│   ├── monitor/
│   │   └── main.go               # App principal: loop continuo
│   └── capture-charts/
│       └── main.go               # App de testing: ejecución única
│
├── pkg/                           # Lógica de negocio reutilizable
│   ├── monitor/
│   │   └── monitoreo.go          # Orquestador principal
│   └── scraper/
│       └── chart_scraper.go      # Scraper de gráficos (chromedp)
│
├── training_data/                 # Datos generados (output)
│   ├── dataset.json              # Dataset completo
│   ├── current_snapshot.json     # Estado actual
│   ├── changes_log.json          # Historial de cambios
│   └── snapshots_YYYYMMDD.json   # Backup diario
│
├── bin/                           # Binarios compilados
│   ├── monitor.exe
│   └── capture-charts.exe
│
├── go.mod                         # Dependencias del proyecto
└── go.sum                         # Checksums de dependencias
```

---

## 🔄 Flujo de Datos

### Ciclo Principal (cada 30 segundos)

```
┌──────────────────────────────────────────────────────────────┐
│ 1. INICIO DEL CICLO                                          │
│    - Timestamp actual                                        │
│    - Contador de verificación                                │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 2. OBTENER ASSIGNMENTS (HTTP GET)                            │
│    URL: http://192.168.121.2/api/api/assignments_list       │
│    Response: []Assignment { Salida, SKU, SorterID }         │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 3. CAPTURAR GRÁFICOS (chromedp)                              │
│    Para cada sorter (1 y 2):                                 │
│      - Abrir http://192.168.121.2/assignment/{id}           │
│      - Esperar 3 segundos (renderizado)                      │
│      - Ejecutar JavaScript para extraer:                     │
│        • SKU completo (h1[0])                                │
│        • Porcentaje (h1[1])                                  │
│    Result: ChartData { SorterID, Percentages, OrderedSKUs }│
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 4. CREAR SNAPSHOT (createSnapshot)                           │
│    Combinar datos:                                           │
│      - Assignments del API                                   │
│      - Porcentajes reales del gráfico                        │
│    Calcular distribuciones:                                  │
│      - Global (promedio entre sorters)                       │
│      - Por Sorter                                            │
│      - Por Salida                                            │
│      - Por Sorter+Salida                                     │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 5. DETECTAR CAMBIOS (hasChanges)                             │
│    Comparar con estado anterior:                             │
│      - ¿Misma cantidad de assignments?                       │
│      - ¿Mismo contenido JSON?                                │
│    Si hay cambios:                                           │
│      - Identificar: Added, Removed, Modified                 │
│      - Registrar en changes_log.json                         │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 6. PERSISTIR DATOS                                           │
│    - Agregar snapshot al dataset                             │
│    - Guardar dataset.json (completo)                         │
│    - Guardar current_snapshot.json (último)                  │
│    - Actualizar last_assignments.json                        │
│    - Backup diario: snapshots_YYYYMMDD.json                  │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 7. MOSTRAR ESTADÍSTICAS (displayStats)                       │
│    - Total snapshots recolectados                            │
│    - Tiempo de ejecución                                     │
│    - Distribución actual por sorter con líneas asignadas     │
│    - Distribución por salida                                 │
│    - Formato: "SKU: X%   L1 L2 L3" (líneas de selladora)    │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
                   [SLEEP 30s]
                         │
                         └─────> REPETIR
```

---

## 🧩 Componentes Principales

### 1. Monitor (`pkg/monitor/monitoreo.go`)

**Responsabilidad:** Orquestar el proceso de recolección de datos

#### Función Principal: `Run()`
```go
func Run()
```
**Descripción:** Loop infinito que ejecuta el ciclo de monitoreo  
**Frecuencia:** 30 segundos (configurable)  
**Pasos:**
1. Fetch assignments del API
2. Crear snapshot con análisis
3. Detectar cambios
4. Persistir datos
5. Mostrar estadísticas

#### Función: `createSnapshot()`
```go
func createSnapshot(timestamp time.Time, assignments []Assignment) DataSnapshot
```
**Descripción:** Genera un snapshot completo del estado actual  
**Input:** Timestamp + lista de assignments  
**Output:** DataSnapshot con:
- Contadores básicos (por sorter, por salida)
- Datos de gráficos (si captureCharts=true)
- Distribuciones de calibres multidimensionales

**Algoritmo:**
```
1. Inicializar snapshot con timestamp y assignments
2. Contar assignments por sorter y salida
3. SI captureCharts está activo:
   a. Llamar chartScraper.ScrapeBothSorters()
   b. Para cada sorter:
      - Guardar porcentajes reales por SKU
      - Mapear SKUs a salidas usando assignments
      - Calcular distribución por sorter+salida
   c. Calcular distribución global (promedio)
4. Retornar snapshot completo
```

#### Función: `detectChanges()`
```go
func detectChanges(old, new []Assignment) ChangeDetail
```
**Descripción:** Identifica diferencias entre dos estados  
**Algoritmo:**
```
1. Crear mapas de referencia (SKU+Sorter+Salida como key)
2. Identificar REMOVED (en old pero no en new)
3. Identificar ADDED (en new pero no en old)
4. Identificar MODIFIED (mismo key, diferente valor)
5. Retornar ChangeDetail con listas de cambios
```

#### Función: `hasChanges()`
```go
func hasChanges(old, new []Assignment) bool
```
**Descripción:** Verifica si hay cambios (comparación rápida)  
**Método:** Serializar ambos a JSON y comparar bytes

---

### 2. ChartScraper (`pkg/scraper/chart_scraper.go`)

**Responsabilidad:** Extraer porcentajes reales desde gráficos HTML

#### Función: `ScrapeAssignment()`
```go
func (cs *ChartScraper) ScrapeAssignment(sorterID int) (*ChartData, error)
```
**Descripción:** Captura datos de un sorter específico  
**Proceso:**
```
1. Construir URL: http://192.168.121.2/assignment/{sorterID}
2. Crear contexto de Chrome con timeout (30s)
3. Navegar a la página
4. Esperar 3 segundos (renderizado JavaScript)
5. Ejecutar script JavaScript:
   - Seleccionar: div.relative.w-full.flex.justify-between.items-center
   - Extraer h1[0] = SKU completo
   - Extraer h1[1] = Porcentaje con %
6. Parsear datos:
   - Remover símbolo %
   - Convertir string a float64
   - Guardar en map[string]float64
7. Retornar ChartData con orden preservado
```

**JavaScript ejecutado:**
```javascript
(() => {
  const data = [];
  const containers = document.querySelectorAll(
    'div.relative.w-full.flex.justify-between.items-center'
  );
  
  containers.forEach(container => {
    const h1Elements = container.querySelectorAll(
      'h1.text-xs.font-bold.px-1.text-center'
    );
    if (h1Elements.length >= 2) {
      const sku = h1Elements[0].textContent.trim();
      const percentage = h1Elements[1].textContent.trim();
      data.push([sku, percentage]);
    }
  });
  
  return data;
})();
```

#### Función: `ScrapeBothSorters()`
```go
func (cs *ChartScraper) ScrapeBothSorters() ([]*ChartData, error)
```
**Descripción:** Captura datos de ambos sorters (1 y 2)  
**Manejo de errores:** Continúa aunque un sorter falle

#### Función: `GetCalibreDistribution()`
```go
func (cd *ChartData) GetCalibreDistribution() map[string]float64
```
**Descripción:** Agrupa porcentajes por tipo de calibre  
**Ejemplo:**
```
Input (SKUs completos):
  "4J-D-SANTINA-C5WFTFG": 26%
  "4J-L-SANTINA-C5CZMG": 10%
  "3J-D-SANTINA-C5WFTFG": 33%

Output (por calibre):
  "Cuadruple_Jumbo": 36%  (26 + 10)
  "Triple_Jumbo": 33%
```

---

### 3. Función: `getSalidasForSKU()` (Nuevo)

```go
func getSalidasForSKU(assignments []Assignment, sorterID int, sku string) string
```
**Responsabilidad:** Obtener las líneas de selladora donde está asignado un SKU  
**Descripción:** Busca todos los assignments de un SKU específico en un sorter y retorna las líneas formateadas

**Algoritmo:**
```
1. Crear lista vacía de salidas
2. FOR cada assignment en assignments:
   SI assignment.SorterID == sorterID Y assignment.SKU == sku:
     SI salida NO está en lista (evitar duplicados):
       Agregar assignment.Salida a lista
3. Ordenar salidas numéricamente
4. Formatear como "L1 L2 L3"
5. RETURN string formateado
```

**Ejemplo de uso:**
```go
assignments := []Assignment{
  {Salida: 2, SKU: "4J-D-LAPINS-C5WFTFG", SorterID: 1},
  {Salida: 7, SKU: "4J-D-LAPINS-C5WFTFG", SorterID: 1},
  {Salida: 3, SKU: "3J-L-LAPINS-C5WFTFG", SorterID: 1},
}

lineas := getSalidasForSKU(assignments, 1, "4J-D-LAPINS-C5WFTFG")
// Output: "L2 L7"
```

**Propósito:** Mostrar al usuario en qué líneas físicas (selladoras) está configurado cada SKU, permitiendo verificar la configuración del sistema.

---

#### Función: `exportToCSV()`
```go
func exportToCSV(snapshot DataSnapshot) error
```

**Descripción:** Exporta los datos de calibres a formato CSV para entrenamiento de modelos ML.

**Input:** DataSnapshot con chart_data y distribuciones  
**Output:** Archivo `training_data/training_data.csv`

**Estructura del CSV:**
```csv
timestamp;sorter_id;sku;calibre;calidad;variedad;lineas;porcentaje;total_skus_activos
2025-11-27 16:53:59;1;4J-D-SANTINA-C5WFTFG;4J;D;SANTINA;L2 L7;26.0;9
2025-11-27 16:53:59;1;3J-D-SANTINA-C5WFTFG;3J;D;SANTINA;L3 L5;33.0;9
```

**Formato del archivo:**
- **Delimitador:** Punto y coma (`;`) para compatibilidad con Excel en español
- **Codificación:** UTF-8
- **Modo:** Append (agrega datos sin borrar registros anteriores)

**Columnas:**
- `timestamp`: Fecha/hora de la captura
- `sorter_id`: ID del sorter (1 o 2)
- `sku`: Código completo del producto
- `calibre`: Calibre de la fruta (extraído del SKU)
- `calidad`: Calidad de la fruta (extraído del SKU)
- `variedad`: Variedad de la fruta (extraído del SKU)
- `lineas`: Líneas de selladora asignadas (ej: "L2 L7")
- `porcentaje`: Porcentaje real del gráfico
- `total_skus_activos`: Total de SKUs activos en ese sorter

**Algoritmo:**
```
1. Verificar si archivo existe (para determinar si escribir headers)
2. Abrir archivo en modo append (O_APPEND | O_CREATE)
3. Configurar writer con delimitador ';'
4. Si archivo es nuevo, escribir headers
5. Para cada sorter con datos de gráfico:
   a. Obtener assignments del sorter
   b. Para cada SKU con porcentaje:
      - Parsear SKU para extraer calibre, calidad, variedad
      - Obtener líneas con getSalidasForSKU() (normalizado)
      - Escribir fila con todos los datos
6. Flush y cerrar archivo
```

**Normalización de SKUs:**
- Los SKUs del gráfico pueden tener diferente capitalización (ej: "Lapins" vs "LAPINS")
- `getSalidasForSKU()` normaliza a mayúsculas para el match correcto
- Garantiza que las líneas se asignen correctamente a cada SKU

**Parseo de SKU:**
```
SKU: "4J-D-SANTINA-C5WFTFG"
  ↓
Calibre: "4J"
Calidad: "D"
Variedad: "SANTINA"
```

**Propósito:** Generar dataset estructurado para entrenar modelos ML que detecten porcentajes de calibres automáticamente.

---

#### Función: `loadConfig()`
```go
func loadConfig()
```

**Descripción:** Carga la configuración desde `config.yaml` y actualiza las variables globales del sistema.

**Algoritmo:**
```
1. Leer archivo config.yaml
2. Parsear YAML a struct Config
3. Actualizar variables globales:
   - baseURL desde config.Packing.URL
   - checkInterval desde config.Monitor.IntervaloSegundos
   - captureCharts desde config.Monitor.CaptureCharts
   - datasetFolder desde config.Data.Folder
4. Recalcular rutas de archivos derivadas
5. Mostrar confirmación con info del packing
```

**Manejo de errores:** Si falla la carga del YAML, usa valores por defecto y continúa.

**Ejemplo de output:**
```
✓ Configuración cargada: Danich Cerezas (cereza) - 2 sorters, 7 líneas
```

---

## 📊 Modelos de Datos

### Config (YAML Structs)

```go
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
```

**Fuente:** Archivo `config.yaml`  
**Propósito:** Configuración flexible del sistema para diferentes packings

---

### Assignment
```go
type Assignment struct {
    Salida   int    `json:"salida"`    // Número de salida (1-7)
    SKU      string `json:"sku"`       // Código del producto
    SorterID int    `json:"sorter_id"` // ID del sorter (1 o 2)
}
```
**Fuente:** API HTTP  
**Ejemplo:**
```json
{
  "salida": 3,
  "sku": "4J-D-SANTINA-C5WFTFG",
  "sorter_id": 1
}
```

---

### ChartData
```go
type ChartData struct {
    SorterID    int                `json:"sorter_id"`
    Timestamp   time.Time          `json:"timestamp"`
    Percentages map[string]float64 `json:"percentages"`   // SKU → %
    OrderedSKUs []string           `json:"ordered_skus"`  // Orden del gráfico
    TotalSKUs   int                `json:"total_skus"`
}
```
**Fuente:** Scraping con chromedp  
**Ejemplo:**
```json
{
  "sorter_id": 1,
  "timestamp": "2025-11-27T11:47:50Z",
  "percentages": {
    "4J-D-SANTINA-C5WFTFG": 26,
    "3J-D-SANTINA-C5WFTFG": 33
  },
  "ordered_skus": [
    "4J-D-SANTINA-C5WFTFG",
    "3J-D-SANTINA-C5CZMG",
    "3J-D-SANTINA-C5WFTFG"
  ],
  "total_skus": 9
}
```

---

### CalibreDistribution
```go
type CalibreDistribution struct {
    Count      int     `json:"count"`      // Cantidad de assignments
    Percentage float64 `json:"percentage"` // Porcentaje real del gráfico
}
```
**Nota importante:** `Count` no se usa actualmente (siempre 0) porque los porcentajes provienen directamente del gráfico, no de conteos manuales.

---

### DataSnapshot
```go
type DataSnapshot struct {
    Timestamp   string                     `json:"timestamp"`
    DateTime    time.Time                  `json:"datetime"`
    Assignments []Assignment               `json:"assignments"`
    TotalCount  int                        `json:"total_count"`
    BySorter    map[int]int                `json:"by_sorter"`
    BySalida    map[int]int                `json:"by_salida"`
    ChartData   map[int]*ChartData         `json:"chart_data,omitempty"`
    
    // Distribuciones multidimensionales
    CalibrePercent        map[string]float64                        `json:"calibre_percent,omitempty"`
    CalibreBySorter       map[int]map[string]CalibreDistribution    `json:"calibre_by_sorter,omitempty"`
    CalibreBySalida       map[int]map[string]CalibreDistribution    `json:"calibre_by_salida,omitempty"`
    CalibreBySorterSalida map[string]map[string]CalibreDistribution `json:"calibre_by_sorter_salida,omitempty"`
}
```

**Estructura de datos:**
```
DataSnapshot
├── timestamp: "2025-11-27 11:47:50"
├── assignments: [...]
├── chart_data:
│   ├── [1]: ChartData del Sorter 1
│   └── [2]: ChartData del Sorter 2
│
├── calibre_percent: (GLOBAL)
│   ├── "4J-D-SANTINA-C5WFTFG": 28%
│   └── "3J-D-SANTINA-C5WFTFG": 35%
│
├── calibre_by_sorter:
│   ├── [1]: (Sorter 1)
│   │   ├── "4J-D-SANTINA-C5WFTFG": {Percentage: 26}
│   │   └── "3J-D-SANTINA-C5WFTFG": {Percentage: 33}
│   └── [2]: (Sorter 2)
│       └── ...
│
├── calibre_by_salida:
│   ├── [1]: (Salida 1)
│   ├── [2]: (Salida 2)
│   └── ...
│
└── calibre_by_sorter_salida:
    ├── "1-1": (Sorter 1, Salida 1)
    ├── "1-2": (Sorter 1, Salida 2)
    └── ...
```

---

### TrainingDataset
```go
type TrainingDataset struct {
    CollectionStart time.Time      `json:"collection_start"`
    CollectionEnd   time.Time      `json:"collection_end"`
    TotalSnapshots  int            `json:"total_snapshots"`
    Snapshots       []DataSnapshot `json:"snapshots"`
}
```
**Propósito:** Contenedor de todos los snapshots recolectados  
**Persistencia:** `training_data/dataset.json`

---

### ChangeLog
```go
type ChangeLog struct {
    Timestamp   string               `json:"timestamp"`
    ChangeType  string               `json:"change_type"`
    Added       []Assignment         `json:"added,omitempty"`
    Removed     []Assignment         `json:"removed,omitempty"`
    Modified    []ModifiedAssignment `json:"modified,omitempty"`
    Description string               `json:"description"`
}
```
**Propósito:** Registrar cambios detectados en assignments  
**Tipos de cambios:**
- `update`: Cambios en assignments existentes
- `initial`: Primera captura del sistema

---

## 🧮 Algoritmos y Lógica

### 1. Detección de Cambios

**Objetivo:** Identificar diferencias entre dos estados de assignments

**Algoritmo:**
```
INPUT: old[]Assignment, new[]Assignment
OUTPUT: ChangeDetail{Added, Removed, Modified}

1. Crear mapa oldMap: key = "SKU-SorterID-Salida" → Assignment
2. Crear mapa newMap: similar

3. DETECTAR REMOVED:
   FOR cada key en oldMap:
     SI key NO existe en newMap:
       Agregar oldMap[key] a lista Removed

4. DETECTAR ADDED y MODIFIED:
   FOR cada key en newMap:
     SI key NO existe en oldMap:
       Agregar newMap[key] a lista Added
     SI NO:
       SI oldMap[key] != newMap[key]:
         Agregar {Old: oldMap[key], New: newMap[key]} a Modified

5. RETURN ChangeDetail{Added, Removed, Modified}
```

**Complejidad:** O(n + m) donde n = len(old), m = len(new)

---

### 2. Extracción de Calibre

**Objetivo:** Extraer el tipo de calibre desde un SKU completo

**Algoritmo:**
```go
func extractCalibre(sku string) string {
    // Caso especial: descarte
    SI sku == "descarte" (case-insensitive):
        RETURN "Descarte"
    
    // Split por guión: "4J-D-SANTINA-C5WFTFG" → ["4J", "D", "SANTINA", "C5WFTFG"]
    parts = sku.split("-")
    calibre = parts[0]
    
    // Mapeo a nombres completos
    SWITCH calibre:
        "J"   → RETURN "Jumbo"
        "2J"  → RETURN "Doble_Jumbo"
        "3J"  → RETURN "Triple_Jumbo"
        "4J"  → RETURN "Cuadruple_Jumbo"
        "XL"  → RETURN "Extra_Large"
        DEFAULT → RETURN calibre (sin cambios)
}
```

**Casos de prueba:**
```
"4J-D-SANTINA-C5WFTFG"  → "Cuadruple_Jumbo"
"J-D-SANTINA-C5CZMG"    → "Jumbo"
"XL-D-SANTINA-C5CZGR"   → "Extra_Large"
"DESCARTE"              → "Descarte"
```

---

### 3. Cálculo de Distribución Global

**Objetivo:** Promediar porcentajes entre sorters para obtener distribución global

**Algoritmo:**
```
INPUT: chartDataList []*ChartData (datos de ambos sorters)
OUTPUT: map[string]float64 (SKU → porcentaje promedio)

1. Inicializar resultado = map vacío

2. AGREGAR datos del Sorter 1:
   FOR cada (sku, percent) en chartDataList[0].Percentages:
     resultado[sku] = percent

3. SI hay Sorter 2 (len(chartDataList) > 1):
   FOR cada (sku, percent) en chartDataList[1].Percentages:
     SI sku existe en resultado:
       resultado[sku] = (resultado[sku] + percent) / 2.0
     SI NO:
       resultado[sku] = percent

4. RETURN resultado
```

**Ejemplo:**
```
Sorter 1: {"4J-D-SANTINA": 30%, "3J-D-SANTINA": 40%}
Sorter 2: {"4J-D-SANTINA": 26%, "3J-D-SANTINA": 38%, "2J-D-SANTINA": 20%}

Resultado:
  "4J-D-SANTINA": (30 + 26) / 2 = 28%
  "3J-D-SANTINA": (40 + 38) / 2 = 39%
  "2J-D-SANTINA": 20%  (solo en Sorter 2)
```

---

### 4. Mapeo de Porcentajes a Salidas

**Objetivo:** Asociar porcentajes del gráfico con salidas físicas

**Algoritmo:**
```
INPUT: 
  - chartData: ChartData (porcentajes por SKU)
  - assignments: []Assignment (asignaciones SKU → Salida)

OUTPUT: map[int]map[string]CalibreDistribution (Salida → SKU → Distribución)

1. Inicializar resultado = map vacío

2. FOR cada assignment en assignments:
   SI assignment.SorterID == chartData.SorterID:
     sku = assignment.SKU
     salida = assignment.Salida
     
     SI sku existe en chartData.Percentages:
       realPercent = chartData.Percentages[sku]
       
       // Crear entrada para esta salida si no existe
       SI resultado[salida] es nil:
         resultado[salida] = map vacío
       
       // Guardar porcentaje real
       resultado[salida][sku] = CalibreDistribution{
         Count: 1,
         Percentage: realPercent
       }

3. RETURN resultado
```

**Concepto clave:** Los porcentajes NO se calculan contando assignments. Se usan los valores **reales del gráfico** que provienen de sensores del sorter.

---

## ⚙️ Configuración

### Sistema de Configuración Flexible (config.yaml)

El sistema utiliza archivos YAML para máxima flexibilidad y adaptación a cualquier packing.

**Archivo:** `config.yaml`
```yaml
# Información del packing (un packing a la vez)
packing:
  name: "Danich Cerezas"
  url: "http://192.168.121.2"
  sorters: 2
  lineas: 7
  fruta: "cereza"

# Configuración del monitoreo
monitor:
  intervalo_segundos: 30
  capture_charts: true

# Rutas de datos
data:
  folder: "training_data"
```

**Filosofía del Sistema:**
- ✅ **Un packing a la vez:** Cada packing tiene su propio servidor y configuración
- ✅ **Completamente maleable:** Solo edita el YAML para cambiar de packing
- ✅ **Sin código hardcoded:** Todas las variables se cargan del YAML
- ✅ **Adaptable:** Funciona con cualquier número de sorters, líneas y tipos de fruta

### Variables Globales (`pkg/monitor/monitoreo.go`)

```go
var (
    // Configuración de red (cargada desde YAML)
    baseURL        string
    assignmentsURL string
    
    // Configuración de monitoreo (cargada desde YAML)
    checkInterval  time.Duration
    captureCharts  bool
    
    // Configuración de persistencia (cargada desde YAML)
    datasetFolder       string
    currentSnapshotFile string
    datasetFile         string
    changesLogFile      string
    lastAssignmentsFile = "last_assignments.json"
)
```

### Modificar configuración:

**Para cambiar de packing**, simplemente edita `config.yaml`:

```yaml
packing:
  name: "Packing XYZ"
  url: "http://192.168.1.100"    # Diferente servidor
  sorters: 3                      # Más sorters
  lineas: 12                      # Más líneas
  fruta: "arandano"               # Diferente fruta

monitor:
  intervalo_segundos: 60          # Monitoreo cada minuto
  capture_charts: false           # Sin captura de gráficos

data:
  folder: "training_data_xyz"     # Carpeta específica
```

**No requiere recompilar el código.** Solo reinicia el monitor.

---

## 🔌 Dependencias Externas

### go.mod completo

```go
module danich

go 1.25.4

require (
    github.com/PuerkitoBio/goquery v1.11.0
    github.com/andybalholm/cascadia v1.3.3
    github.com/chromedp/cdproto v0.0.0-20250803210736-d308e07a266d
    github.com/chromedp/chromedp v0.14.2
    github.com/chromedp/sysutil v1.1.0
    github.com/go-json-experiment/json v0.0.0-20251027170946-4849db3c2f7e
    github.com/gobwas/httphead v0.1.0
    github.com/gobwas/pool v0.2.1
    github.com/gobwas/ws v1.4.0
    golang.org/x/net v0.47.0
    golang.org/x/sys v0.38.0
)
```

### Descripción de dependencias:

| Paquete | Versión | Propósito |
|---------|---------|-----------|
| `chromedp/chromedp` | v0.14.2 | Control de Chrome headless |
| `chromedp/cdproto` | latest | Protocolo DevTools de Chrome |
| `chromedp/sysutil` | v1.1.0 | Utilidades del sistema para chromedp |
| `PuerkitoBio/goquery` | v1.11.0 | Parsing HTML (estilo jQuery) |
| `andybalholm/cascadia` | v1.3.3 | Selectores CSS (usado por goquery) |
| `gobwas/ws` | v1.4.0 | WebSocket client (para chromedp) |
| `golang.org/x/net` | v0.47.0 | Extensiones de red de Go |
| `golang.org/x/sys` | v0.38.0 | Llamadas al sistema |

### Instalación de dependencias:

```bash
go mod download
```

---

## 📋 Casos de Uso

### Caso 1: Monitoreo Continuo

**Actor:** Sistema automatizado  
**Objetivo:** Recolectar datos cada 30 segundos para ML

**Flujo:**
```
1. Usuario ejecuta: ./bin/monitor.exe
2. Sistema inicializa:
   - Carga dataset existente
   - Inicializa ChartScraper
3. LOOP infinito:
   a. Obtener assignments del API
   b. Capturar gráficos de ambos sorters
   c. Crear snapshot con distribuciones
   d. Detectar cambios vs estado anterior
   e. Guardar en dataset.json
   f. Mostrar estadísticas
   g. Sleep 30 segundos
4. Usuario detiene con Ctrl+C
```

**Output generado:**
- `training_data/dataset.json` (creciente)
- `training_data/current_snapshot.json` (actualizado)
- `training_data/changes_log.json` (append)
- `training_data/snapshots_YYYYMMDD.json` (backup diario)
- `training_data/training_data.csv` (dataset para ML)

---

### Caso 2: Captura Única para Testing

**Actor:** Desarrollador  
**Objetivo:** Probar el scraper sin loop continuo

**Flujo:**
```
1. Desarrollador ejecuta: ./bin/capture-charts.exe
2. Sistema:
   a. Crea ChartScraper
   b. Captura datos de Sorter 1
   c. Captura datos de Sorter 2
   d. Muestra resultados en consola
   e. Guarda en chart_data_captured.json
3. Termina la ejecución
```

**Output:**
```
=== Capturando datos de gráficos de ambos sorters ===

Sorter: 1
Timestamp: 2025-11-27 11:47:50
Total SKUs: 9
Detalle por SKU:
  4J-D-SANTINA-C5WFTFG: 26%   L2 L7
  3J-D-SANTINA-C5WFTFG: 33%   L3 L5
  ...

✓ Datos guardados en: chart_data_captured.json
```

---

### Caso 3: Análisis de Cambios

**Actor:** Sistema  
**Objetivo:** Detectar cuándo cambian las asignaciones

**Flujo:**
```
1. Sistema tiene estado previo en last_assignments.json
2. Obtiene nuevo estado del API
3. Ejecuta hasChanges():
   - Serializa ambos estados a JSON
   - Compara bytes
4. SI hay cambios:
   a. Ejecuta detectChanges()
   b. Identifica: Added, Removed, Modified
   c. Muestra en consola:
      🔔 ¡CAMBIOS DETECTADOS!
      + Agregados: 2 assignments
      - Eliminados: 1 assignment
      ≈ Modificados: 3 assignments
   d. Guarda en changes_log.json
5. SI NO hay cambios:
   Muestra: ✓ Sin cambios
```

---

## 🔍 Puntos Clave para Estudio

### 1. ¿Por qué chromedp en lugar de HTTP simple?

**Respuesta:** Los gráficos son generados con JavaScript (probablemente React/Vue). El HTML inicial no contiene los porcentajes. chromedp:
- Ejecuta el JavaScript
- Espera el renderizado
- Lee el DOM final

### 2. ¿Por qué guardar OrderedSKUs?

**Respuesta:** El orden en el gráfico tiene significado (probablemente ordenado por importancia o frecuencia). Preservarlo permite:
- Mostrar datos igual que el gráfico original
- Análisis de patrones de ordenamiento
- Debugging (comparar con pantalla real)

### 3. ¿Por qué Count siempre es 0 en CalibreDistribution?

**Respuesta:** Inicialmente se calculaban porcentajes contando assignments. Luego se descubrió que no coincidían con los porcentajes reales del gráfico (que provienen de sensores). Ahora:
- `Percentage` = dato real del gráfico ✅
- `Count` = obsoleto (se mantiene por compatibilidad JSON)

### 4. ¿Por qué hay doble `/api/` en la URL?

**Respuesta:** Es una peculiaridad del backend existente:
```
http://192.168.121.2/api/api/assignments_list
                     ^   ^
                     1   2
```
Probablemente un prefijo de ruta en el router + endpoint específico.

### 5. ¿Cómo se mapean los SKUs a salidas?

**Respuesta:** El API devuelve assignments que dicen "este SKU debe ir a esta salida en este sorter". El gráfico dice "este SKU está saliendo al X%". Combinándolos sabemos:
```
Assignment 1: "4J-D-SANTINA-C5WFTFG" → Salida 3, Sorter 1
Assignment 2: "4J-D-SANTINA-C5WFTFG" → Salida 5, Sorter 1
ChartData:    "4J-D-SANTINA-C5WFTFG" → 26%
Conclusión:   Este SKU está en Salidas 3 y 5 con 26% total
Display:      "4J-D-SANTINA-C5WFTFG: 26%   L3 L5"
```

**Nota importante:** Un mismo SKU puede estar asignado a **múltiples líneas/salidas** en el mismo sorter. Esto permite distribuir un producto en diferentes selladoras según la configuración de la planta.

---

## 🚀 Compilación y Despliegue

### Compilar para Windows

```bash
# Monitor principal
GOOS=windows GOARCH=amd64 go build -o bin/monitor.exe cmd/monitor/main.go

# Herramienta de testing
GOOS=windows GOARCH=amd64 go build -o bin/capture-charts.exe cmd/capture-charts/main.go
```

### Compilar para Linux

```bash
GOOS=linux GOARCH=amd64 go build -o bin/monitor cmd/monitor/main.go
```

### Ejecutar con variables de entorno

```bash
# Cambiar URL sin modificar código
BASE_URL="http://192.168.1.100" ./bin/monitor.exe
```

---

## 📚 Referencias

### Documentación oficial:

- **Go:** https://go.dev/doc/
- **chromedp:** https://github.com/chromedp/chromedp
- **goquery:** https://github.com/PuerkitoBio/goquery

### Conceptos Go importantes:

- **Goroutines y Concurrencia:** https://go.dev/tour/concurrency
- **Context:** https://go.dev/blog/context
- **JSON encoding:** https://go.dev/blog/json

### Chrome DevTools Protocol:

- **Documentación:** https://chromedevtools.github.io/devtools-protocol/

---

## 🧪 Testing y Debugging

### Probar scraper sin monitor:

```bash
./bin/capture-charts.exe
```

### Verificar conectividad con API:

```bash
curl http://192.168.121.2/api/api/assignments_list
```

### Ver logs de chromedp:

```go
// Agregar en chart_scraper.go
ctx, cancel = chromedp.NewContext(ctx, chromedp.WithDebugf(log.Printf))
```

### Validar JSON generado:

```bash
cat training_data/dataset.json | jq '.snapshots | length'
```

---

## 🎓 Resumen para Estudio

### Conceptos clave:

1. **Web Scraping con JavaScript:** chromedp permite interactuar con SPAs
2. **Arquitectura Hexagonal:** Separación clara entre negocio e infraestructura
3. **Persistencia JSON:** Formato legible para análisis posterior
4. **Event Detection:** Comparación de estados para detectar cambios
5. **Data Aggregation:** Múltiples dimensiones de análisis (sorter, salida, combinado)
6. **Multi-line Assignment:** Un SKU puede estar en múltiples líneas de selladora simultáneamente

### Flujo de aprendizaje sugerido:

1. ✅ Entender el propósito del sistema
2. ✅ Leer `cmd/monitor/main.go` (punto de entrada)
3. ✅ Estudiar `pkg/monitor/monitoreo.go` (lógica principal)
4. ✅ Analizar `pkg/scraper/chart_scraper.go` (scraping)
5. ✅ Revisar modelos de datos
6. ✅ Ejecutar y observar outputs
7. ✅ Modificar configuraciones
8. ✅ Agregar features propios


## 🌐 Escalabilidad y Filosofía

### Diseño Multi-Packing

El sistema está diseñado con la filosofía de **"un packing a la vez"** pero con **máxima flexibilidad**:

**Principios:**
1. **Cada packing tiene su propio servidor:** Diferentes URLs, redes locales
2. **Configuración independiente:** Cada packing define sorters, líneas, fruta
3. **Cambio rápido:** Solo editar YAML y reiniciar (no recompilar)
4. **Sin código hardcoded:** Todo configurable externamente

**¿Por qué un packing a la vez?**
- Simplicidad operacional
- Enfoque claro en un contexto
- Facilita debugging y análisis
- Los packings suelen estar en redes diferentes

**Adaptación a nuevos packings:**
```yaml
# Ejemplo: Packing en otra planta
packing:
  name: "Packing ABC"
  url: "http://10.0.0.50"    # Diferente servidor
  sorters: 3                  # Más sorters
  lineas: 12                  # Más líneas
  fruta: "arandano"           # Diferente fruta
```

El sistema se adapta automáticamente sin modificar código.

**Archivos clave para portabilidad:**
- `config.yaml`: Configuración activa
- `config.example.yaml`: Ejemplos de diferentes packings
- Los binarios compilados (`bin/monitor.exe`) son portables

---

**Autor:** Sistema MonitoreoDanich  
**Última actualización:** 27 Noviembre 2025  
**Versión:** 1.2
