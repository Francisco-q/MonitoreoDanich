# Documentación Técnica - Sistema de Monitoreo y Advisor de Sorters

**Proyecto:** MonitoreoDanich  
**Propósito:** Sistema completo de monitoreo continuo, recolección de datos, inferencia automática de decisiones y asesoramiento inteligente con IA para optimización de sorters de fruta  
**Fecha:** Noviembre 2025  
**Versión:** 2.0  
**Lenguajes:** Go 1.25.4 + Python 3.11

---

## 📑 Tabla de Contenidos

1. [Visión General del Sistema](#visión-general-del-sistema)
2. [Arquitectura General](#arquitectura-general)
3. [Stack Tecnológico](#stack-tecnológico)
4. [Sistema de Monitoreo](#sistema-de-monitoreo)
5. [Sistema de Advisor (IA)](#sistema-de-advisor-ia)
6. [Sistema de Inferencia Automática](#sistema-de-inferencia-automática)
7. [Machine Learning y Entrenamiento](#machine-learning-y-entrenamiento)
8. [Modelos de Datos](#modelos-de-datos)
9. [Configuración y Despliegue](#configuración-y-despliegue)
10. [Comandos y Herramientas](#comandos-y-herramientas)
11. [Datos Generados y Persistencia](#datos-generados-y-persistencia)
12. [Guía de Uso y Workflows](#guía-de-uso-y-workflows)

---

## 🎯 Visión General del Sistema

### ¿Qué hace este sistema?

MonitoreoDanich es una **plataforma completa de optimización inteligente** para sorters de fruta que combina:

1. **Monitoreo continuo** de sorters en tiempo real (cada 30 segundos)
2. **Recolección automática** de porcentajes reales desde gráficos HTML/JavaScript
3. **Inferencia automática** de decisiones a partir de cambios históricos
4. **Asesoramiento con IA** mediante LLM (Ollama) para balanceo de sorters
5. **Machine Learning** para predecir razones de decisiones operativas

### Dos Enfoques de IA

El sistema implementa **dos estrategias complementarias** de inteligencia artificial:

#### 1. **Prompting con LLM (Ollama)** - "Don Sergio Virtual"
- **Qué hace:** Analiza el estado actual y sugiere movimientos de SKUs entre sorters
- **Cómo funciona:** Usa un modelo de lenguaje (phi3:mini o llama3.2:3b) con prompt engineering
- **Ventaja:** No requiere datos históricos, funciona desde el día 1
- **Uso:** Consulta en tiempo real cuando necesitas asesoramiento

#### 2. **Fine-tuning con XGBoost** - "Aprendizaje del pasado"
- **Qué hace:** Aprende patrones de decisiones históricas y predice razones de cambios
- **Cómo funciona:** Entrena un modelo supervisado con decisiones inferidas automáticamente
- **Ventaja:** Detecta patrones que no son obvios, mejora con más datos
- **Uso:** Análisis predictivo de qué tipo de desbalance causó una decisión

### Flujo Completo del Sistema

```
┌──────────────────┐
│   MONITOREO      │  Captura estado cada 30s
│   (monitor cmd)  │  - API assignments
└────────┬─────────┘  - Gráficos HTML
         │
         ▼
┌──────────────────┐
│  PERSISTENCIA    │  Guarda snapshots + changes
│  (dataset.json)  │  - Estado completo
└────────┬─────────┘  - Historial de cambios
         │
         ▼
┌──────────────────┐
│   INFERENCIA     │  Analiza cambios históricos
│  (infer cmd)     │  - Detecta movimientos de SKUs
└────────┬─────────┘  - Infiere razones automáticamente
         │
         ├─────────────────────┐
         │                     │
         ▼                     ▼
┌──────────────────┐  ┌──────────────────┐
│   PROMPTING      │  │  FINE-TUNING     │
│  (Ollama LLM)    │  │  (train_model)   │
│                  │  │                  │
│ "¿Qué debo       │  │ "¿Por qué se     │
│  hacer ahora?"   │  │  tomó esta       │
│                  │  │  decisión?"      │
└──────────────────┘  └──────────────────┘
```

---

## 🏗️ Arquitectura General

### Patrón de Diseño
**Hexagonal Architecture + Event-Driven Design + AI-Assisted Decision Making + Modular Components**

### Arquitectura Modular (Diciembre 2024)

El sistema ha sido refactorizado de un monolito de **800+ líneas** a una arquitectura modular con **9 componentes especializados**:

#### Beneficios de la Modularización
- ✅ **Mantenibilidad:** Cada archivo tiene una responsabilidad única (SRP)
- ✅ **Testabilidad:** Componentes aislados fáciles de probar
- ✅ **Legibilidad:** Archivos pequeños (1-5KB) vs monolito (24KB)
- ✅ **Escalabilidad:** Agregar features sin tocar código existente
- ✅ **Reutilización:** Componentes independientes reutilizables

#### Estructura Modular

```
pkg/monitor/
├── monitor.go       (5.3KB)  → Orquestador principal con DI
├── models.go        (2.6KB)  → Estructuras de datos compartidas
├── config.go        (3.3KB)  → Carga y parseo de config.yaml
├── fetcher.go       (1.1KB)  → Cliente HTTP para API
├── snapshot.go      (4.5KB)  → Construcción de snapshots
├── changes.go       (2.9KB)  → Detección y análisis de cambios
├── persistence.go   (3.4KB)  → I/O de archivos JSON
├── exporter.go      (3.8KB)  → Exportación a CSV para ML
└── display.go       (3.4KB)  → Visualización en consola

Total: 30.3KB en 9 archivos (vs 24KB en 1 archivo monolítico)
```

#### Flujo de Datos Entre Módulos

```
     ┌──────────────┐
     │  monitor.go  │  ← Punto de entrada, orquesta todo
     └──────┬───────┘
            │
            ├─→ config.go       → Carga configuración
            ├─→ fetcher.go      → Obtiene assignments (API)
            ├─→ snapshot.go     → Crea snapshot (+ scraper)
            ├─→ changes.go      → Detecta diferencias
            ├─→ persistence.go  → Guarda datos (JSON)
            ├─→ exporter.go     → Exporta a CSV
            └─→ display.go      → Muestra en consola
```

#### Inyección de Dependencias

```go
// Constructor con DI
m := &Monitor{
    config:         LoadConfig(),              // config.go
    fetcher:        NewFetcher(url),           // fetcher.go
    persistence:    NewPersistence(config),    // persistence.go
    changeDetector: NewChangeDetector(),       // changes.go
    snapshotBuilder: NewSnapshotBuilder(scraper), // snapshot.go
    exporter:       NewExporter(folder),       // exporter.go
    display:        NewDisplay(config),        // display.go
}
```

**Ventaja:** Cada componente recibe solo lo que necesita, sin acoplamiento global.

```
┌─────────────────────────────────────────────────────────────────┐
│                      APLICACIONES (cmd/)                        │
│                                                                 │
│  ┌──────────────┐  ┌───────────────┐  ┌────────────────────┐  │
│  │   monitor    │  │ capture-charts│  │ infer-decisions    │  │
│  │   (loop)     │  │   (testing)   │  │  (inference)       │  │
│  └──────────────┘  └───────────────┘  └────────────────────┘  │
└────────┬────────────────────┬─────────────────────┬────────────┘
         │                    │                     │
         ▼                    ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                   LÓGICA DE NEGOCIO (pkg/)                      │
│                                                                 │
│  ┌────────────────────────────────────────────────────┐        │
│  │  monitor/ (ARQUITECTURA MODULAR - 9 componentes)  │        │
│  │  ┌───────────────────────────────────────────┐    │        │
│  │  │ monitor.go → Orquestador (DI + Run loop) │    │        │
│  │  └───────────────────────────────────────────┘    │        │
│  │  ┌───────────────────────────────────────────┐    │        │
│  │  │ models.go → Estructuras de datos         │    │        │
│  │  │ config.go → Carga YAML                   │    │        │
│  │  │ fetcher.go → Cliente HTTP API            │    │        │
│  │  │ snapshot.go → Construcción snapshots     │    │        │
│  │  │ changes.go → Detección de cambios        │    │        │
│  │  │ persistence.go → I/O JSON                │    │        │
│  │  │ exporter.go → Exportación CSV            │    │        │
│  │  │ display.go → Visualización consola       │    │        │
│  │  └───────────────────────────────────────────┘    │        │
│  └────────────────────────────────────────────────────┘        │
│                                                                 │
│  ┌────────────────────────────────────────────────────┐        │
│  │  scraper/chart_scraper.go                          │        │
│  │  - ScrapeAssignment()  (chromedp un sorter)        │        │
│  │  - ScrapeBothSorters() (ambos sorters)             │        │
│  └────────────────────────────────────────────────────┘        │
│                                                                 │
│  ┌────────────────────────────────────────────────────┐        │
│  │  advisor/advisor.go (IA - Prompting)               │        │
│  │  - GetAdvice()     (consulta Ollama LLM)           │        │
│  └────────────────────────────────────────────────────┘        │
│                                                                 │
│  ┌────────────────────────────────────────────────────┐        │
│  │  advisor/decision_inference.go (ML - Training)     │        │
│  │  - InferDecisionsFromChanges() (análisis histórico)│        │
│  │  - analyzeChange()    (detecta movimientos SKU)    │        │
│  │  - inferReason()      (clasifica tipo cambio)      │        │
│  │  - calculateConfidence() (score 0-1)               │        │
│  └────────────────────────────────────────────────────┘        │
└────────┬──────────────────────┬────────────────┬───────────────┘
         │                      │                │
         ▼                      ▼                ▼
┌─────────────────────────────────────────────────────────────────┐
│                       INFRAESTRUCTURA                           │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │  HTTP API    │  │  Chromedp    │  │  Ollama (LLM)      │   │
│  │ 192.168.*    │  │  (Browser)   │  │  localhost:11434   │   │
│  └──────────────┘  └──────────────┘  └────────────────────┘   │
│                                                                 │
│  ┌──────────────────────────────────────────────────────┐      │
│  │  Filesystem (JSON + CSV)                             │      │
│  │  - dataset.json        (snapshots históricos)        │      │
│  │  - changes_log.json    (cambios detectados)          │      │
│  │  - training_data.csv   (dataset ML)                  │      │
│  │  - decisiones_inferidas.json (decisions auto)        │      │
│  │  - decisiones_training.csv   (para XGBoost)          │      │
│  └──────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      CAPA DE MACHINE LEARNING                   │
│                                                                 │
│  ┌────────────────────────────────────────────────────┐        │
│  │  train_model.py (Python 3.11)                      │        │
│  │  - XGBoost Classifier                              │        │
│  │  - 18 features (base + derived)                    │        │
│  │  - 6 clases de decisión                            │        │
│  │  - One-hot encoding para calibres                  │        │
│  │  - Output: decision_model.pkl                      │        │
│  └────────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

### Principios Aplicados

1. **Separation of Concerns**: Cada paquete tiene una responsabilidad única
2. **Dependency Injection**: ChartScraper se inyecta, sin dependencias cíclicas
3. **Single Responsibility**: Funciones pequeñas y enfocadas
4. **Data Persistence**: Estado completo guardado para análisis retrospectivo
5. **No Cyclic Dependencies**: Advisor usa tipos locales, no importa monitor
6. **AI-First Design**: Dos estrategias de IA complementarias (prompting + fine-tuning)

---

## 💻 Stack Tecnológico

### Backend - Go 1.25.4

**¿Por qué Go?**
- ✅ Concurrencia nativa (goroutines) para scraping paralelo
- ✅ Binarios compilados portables sin dependencias externas
- ✅ Alto rendimiento para operaciones de red
- ✅ Excelente para herramientas CLI y daemons

### Machine Learning - Python 3.11

**¿Por qué Python?**
- ✅ Ecosistema ML completo (scikit-learn, XGBoost)
- ✅ Pandas para manipulación de datos
- ✅ Integración directa con el pipeline de Go (CSV/JSON)
- ✅ Comunidad y bibliotecas maduras

### Librerías Go

#### 1. **chromedp** v0.14.2
```go
import "github.com/chromedp/chromedp"
```
**Propósito:** Control headless de Chrome para scraping de JavaScript  
**Uso en el proyecto:**
- Navegar a páginas HTML dinámicas (React/Vue)
- Ejecutar JavaScript y esperar renderizado
- Extraer porcentajes desde el DOM final

**Ejemplo clave:**
```go
chromedp.Run(ctx,
    chromedp.Navigate(url),
    chromedp.Sleep(3*time.Second), // Esperar renderizado
    chromedp.Evaluate(`
        (() => {
          const containers = document.querySelectorAll('.percentage-container');
          return Array.from(containers).map(c => [
            c.querySelector('h1:nth-child(1)').textContent, // SKU
            c.querySelector('h1:nth-child(2)').textContent  // Porcentaje
          ]);
        })()
    `, &resultado),
)
```

#### 2. **goquery** v1.11.0
```go
import "github.com/PuerkitoBio/goquery"
```
**Propósito:** Parsear y consultar documentos HTML (sintaxis jQuery)  
**Uso:** Análisis de estructura HTML estático (si fuera necesario)

#### 3. **yaml.v3**
```go
import "gopkg.in/yaml.v3"
```
**Propósito:** Configuración flexible del sistema  
**Uso:** Cargar `config.yaml` para adaptarse a diferentes packings sin recompilar

#### 4. **Librería Estándar de Go**
```go
"encoding/json"    // Serialización JSON (persistencia)
"encoding/csv"     // Exportación CSV para ML
"net/http"         // Cliente HTTP (API + Ollama)
"time"             // Manejo de timestamps
"io/ioutil"        // I/O de archivos
"context"          // Timeouts y cancelación
"sort"             // Ordenamiento de datos
"strings"          // Normalización de SKUs
"math"             // Cálculos de confianza
```

### Librerías Python

#### 1. **pandas** 2.3.3
```python
import pandas as pd
```
**Propósito:** Manipulación y limpieza de datasets  
**Uso:**
- Cargar CSV semicolon-separated
- Feature engineering (derived features)
- Manejo de datos faltantes

#### 2. **scikit-learn** 1.7.2
```python
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import LabelEncoder
from sklearn.metrics import classification_report, accuracy_score
```
**Propósito:** Pipeline de ML clásico  
**Uso:**
- Train/test split
- One-hot encoding de calibres
- Métricas de evaluación

#### 3. **XGBoost** 3.1.2
```python
from xgboost import XGBClassifier
```
**Propósito:** Modelo de clasificación con gradient boosting  
**Uso:**
- Clasificar tipo de decisión (6 clases)
- Predecir razón del movimiento de SKUs
- Feature importance analysis

#### 4. **joblib** 1.5.2
```python
import joblib
```
**Propósito:** Serialización de modelos ML  
**Uso:** Guardar/cargar `decision_model.pkl` y `label_encoder.pkl`

### IA y LLM

#### **Ollama** (Local LLM Server)
```
Modelo usado: phi3:mini (o llama3.2:3b)
Endpoint: http://localhost:11434/api/generate
```
**Propósito:** Asesoramiento en tiempo real mediante prompting  
**Uso:**
- Analizar estado actual de sorters
- Sugerir movimientos de SKUs
- Responder en formato JSON estructurado

**Ventaja:** No requiere GPU dedicada, corre en CPU local

---

## 📦 Estructura del Proyecto

```
MonitoreoDanich/
│
├── cmd/                                 # Puntos de entrada (executables)
│   ├── monitor/
│   │   └── main.go                     # Loop principal de monitoreo (30s)
│   ├── capture-charts/
│   │   └── main.go                     # Testing: captura única
│   ├── analyze-data/
│   │   └── main.go                     # Análisis de calidad de datos
│   └── infer-decisions/
│       └── main.go                     # **NUEVO** Inferencia automática de decisiones
│
├── pkg/                                 # Lógica de negocio
│   ├── monitor/                        # **MODULARIZADO** - Sistema de monitoreo
│   │   ├── monitor.go                  # Orquestador principal del sistema
│   │   │   - Monitor struct            # Coordina todos los componentes
│   │   │   - New()                     # Constructor con inyección de dependencias
│   │   │   - Run()                     # Loop principal
│   │   │   - runCycle()                # Un ciclo de monitoreo
│   │   │
│   │   ├── models.go                   # Estructuras de datos
│   │   │   - Assignment                # Asignación SKU → Salida
│   │   │   - DataSnapshot              # Estado completo del sistema
│   │   │   - CalibreDistribution       # Distribución de SKUs
│   │   │   - TrainingDataset           # Dataset histórico
│   │   │   - ChangeLog                 # Log de cambios
│   │   │
│   │   ├── config.go                   # Configuración YAML
│   │   │   - LoadConfig()              # Carga y parsea config.yaml
│   │   │   - SystemConfig struct       # Configuración del sistema
│   │   │   - PackingConfig, MonitorConfig, DataConfig
│   │   │
│   │   ├── fetcher.go                  # Cliente HTTP para API
│   │   │   - FetchAssignments()        # GET assignments desde API
│   │   │
│   │   ├── snapshot.go                 # Creación de snapshots
│   │   │   - SnapshotBuilder           # Constructor de snapshots
│   │   │   - CreateSnapshot()          # Genera snapshot completo
│   │   │   - captureChartData()        # Integra datos de gráficos
│   │   │   - ExtractCalibre()          # Normalización de nombres
│   │   │
│   │   ├── changes.go                  # Detección de cambios
│   │   │   - HasChanges()              # Comparación rápida (bytes)
│   │   │   - DetectChanges()           # Análisis detallado
│   │   │   - DisplayChanges()          # Output consola
│   │   │
│   │   ├── persistence.go              # Persistencia de datos
│   │   │   - SaveDataset()             # Guarda dataset.json
│   │   │   - LoadOrCreateDataset()     # Carga o inicializa
│   │   │   - SaveSnapshot()            # Guarda snapshot actual
│   │   │   - LogChange()               # Registra cambios
│   │   │
│   │   ├── exporter.go                 # Exportación a CSV
│   │   │   - ExportToCSV()             # Genera training_data.csv
│   │   │   - createCSVRecord()         # Crea registro por SKU
│   │   │
│   │   └── display.go                  # Visualización en consola
│   │       - ShowStats()               # Estadísticas del sistema
│   │       - ShowAdvice()              # Muestra consejos del advisor
│   │
│   ├── scraper/
│   │   └── chart_scraper.go            # Scraper de gráficos HTML/JS
│   │       - ScrapeAssignment()        # Un sorter
│   │       - ScrapeBothSorters()       # Ambos sorters
│   │
│   └── advisor/                        # **NUEVO** Sistema de IA
│       ├── advisor.go                  # Prompting con Ollama LLM
│       │   - GetAdvice()               # Consulta al modelo
│       │   - tipos locales (sin dependencias cíclicas)
│       │
│       └── decision_inference.go       # Inferencia automática para ML
│           - InferDecisionsFromChanges() # Análisis histórico
│           - analyzeChange()           # Detecta movimientos
│           - inferReason()             # Clasifica tipo (6 categorías)
│           - calculateConfidence()     # Score 0-1
│
├── training_data/                       # Output: datos generados
│   ├── dataset.json                    # Snapshots históricos completos
│   ├── current_snapshot.json           # Estado actual (último)
│   ├── changes_log.json                # Log de cambios detectados
│   ├── snapshots_YYYYMMDD.json         # Backup diario
│   │
│   ├── training_data.csv               # **Dataset para ML** (semicolon CSV)
│   │   # Columnas: timestamp, sorter_id, sku, calibre, calidad,
│   │   #          variedad, lineas, porcentaje, total_skus_activos
│   │
│   ├── decisiones_inferidas.json       # **NUEVO** Decisiones inferidas
│   │   # Con: movimientos, porcentajes antes/después, razón, confianza
│   │
│   └── decisiones_training.csv         # **NUEVO** Dataset para XGBoost
│       # Columnas: sku, calibre, variedad, de_sorter, a_sorter,
│       #          porcentajes antes/después, impacto, razón, confianza
│
├── bin/                                 # Binarios compilados
│   ├── monitor.exe                     # Monitoreo continuo
│   ├── capture-charts.exe              # Testing
│   ├── analyze-data.exe                # Análisis de datos
│   └── infer-decisions.exe             # **NUEVO** Generar decisiones
│
├── venv/                                # **NUEVO** Entorno virtual Python
│   ├── Scripts/
│   │   ├── activate                    # Activación (bash)
│   │   └── python.exe                  # Intérprete Python 3.11
│   └── Lib/                            # Librerías Python
│
├── train_model.py                       # **NUEVO** Script de entrenamiento ML
│   # XGBoost para clasificar tipo de decisión
│   # Input: decisiones_training.csv
│   # Output: decision_model.pkl, label_encoder.pkl, training_metrics.json
│
├── decision_model.pkl                   # **NUEVO** Modelo entrenado (cuando hay datos)
├── label_encoder.pkl                    # **NUEVO** Encoder de etiquetas
├── training_metrics.json                # **NUEVO** Métricas del modelo
│
├── config.yaml                          # Configuración activa
├── go.mod                               # Dependencias Go
├── go.sum                               # Checksums
│
├── DOCUMENTACION_TECNICA.md             # Este archivo
├── FLUJO_DATOS.md                       # Flujo detallado de datos
├── README.md                            # Guía rápida
└── todo.md                              # TODOs y mejoras futuras
```

### Archivos Clave por Función

#### Monitoreo
- `cmd/monitor/main.go` → Punto de entrada
- `pkg/monitor/monitoreo.go` → Lógica completa
- `pkg/scraper/chart_scraper.go` → Captura de gráficos

#### Advisor (IA)
- `pkg/advisor/advisor.go` → Prompting con Ollama
- `pkg/advisor/decision_inference.go` → Inferencia automática
- `cmd/infer-decisions/main.go` → Generador de dataset

#### Machine Learning
- `train_model.py` → Script de entrenamiento
- `decisiones_training.csv` → Dataset de entrada
- `decision_model.pkl` → Modelo entrenado

#### Configuración
- `config.yaml` → Configuración del sistema
- `.env` → Variables de entorno (legacy, opcional)

---

## 🔄 Sistema de Monitoreo

### Ciclo Principal (cada 30 segundos)

```
┌──────────────────────────────────────────────────────────────┐
│ 1. INICIO DEL CICLO                                          │
│    - Timestamp actual                                        │
│    - Contador de verificación                                │
│    - Carga config.yaml si cambió                            │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 2. OBTENER ASSIGNMENTS (HTTP GET)                            │
│    URL: http://192.168.121.2/api/api/assignments_list       │
│    Response: []Assignment { Salida, SKU, SorterID }         │
│    Ejemplo: [                                                │
│      {"salida": 2, "sku": "4J-D-LAPINS-C5WFTFG", "sorter_id": 1}, │
│      {"salida": 7, "sku": "4J-D-LAPINS-C5WFTFG", "sorter_id": 1}, │
│      ...                                                     │
│    ]                                                         │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 3. CAPTURAR GRÁFICOS (chromedp)                              │
│    Para cada sorter (1 y 2):                                 │
│      a) Navegar a: http://192.168.121.2/assignment/{id}     │
│      b) Esperar 3s para renderizado JavaScript               │
│      c) Ejecutar script para extraer DOM:                    │
│         document.querySelectorAll(                           │
│           'div.relative.w-full.flex.justify-between...'      │
│         )                                                    │
│      d) Para cada container:                                 │
│         - h1[0] = SKU completo ("4J-D-LAPINS-C5WFTFG")      │
│         - h1[1] = Porcentaje ("26%")                        │
│      e) Parsear: remover %, convertir a float64             │
│    Result: ChartData {                                       │
│      SorterID: 1,                                            │
│      Percentages: {"4J-D-LAPINS": 26.0, "3J-D-LAPINS": 33.0},│
│      OrderedSKUs: ["4J-D-LAPINS", "3J-D-LAPINS", ...],      │
│      TotalSKUs: 11                                           │
│    }                                                         │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 4. CREAR SNAPSHOT (createSnapshot)                           │
│    Combinar datos:                                           │
│      - Assignments del API (configuración)                   │
│      - Porcentajes reales del gráfico (sensores)            │
│    Normalizar SKUs: ToUpper() para matching                  │
│    Calcular distribuciones multidimensionales:               │
│      - Global: promedio entre sorters                        │
│      - Por Sorter: datos de cada sorter                      │
│      - Por Salida: mapeando SKU → Salida                    │
│      - Por Sorter+Salida: combinación                        │
│    Generar DataSnapshot completo con timestamp               │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 5. DETECTAR CAMBIOS (hasChanges + detectChanges)            │
│    Comparar con estado anterior (last_assignments.json):     │
│      - Serializar ambos a JSON                               │
│      - Comparar bytes (rápido)                               │
│    SI hay cambios:                                           │
│      a) Ejecutar detectChanges() detallado:                  │
│         - Crear mapas: key = "SKU-SorterID-Salida"          │
│         - Identificar REMOVED (en old, no en new)            │
│         - Identificar ADDED (en new, no en old)              │
│         - Identificar MODIFIED (mismo key, diff value)       │
│      b) Registrar en changes_log.json con:                   │
│         - Timestamp                                          │
│         - ChangeType: "update" o "initial"                   │
│         - Listas: Added, Removed, Modified                   │
│         - Description generada                               │
│      c) Mostrar en consola: 🔔 CAMBIOS DETECTADOS           │
│    SI NO hay cambios:                                        │
│      Mostrar: ✓ Sin cambios en assignments                  │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 6. EXPORTAR A CSV (exportToCSV)                              │
│    Para cada sorter con chart_data:                          │
│      Para cada SKU con porcentaje:                           │
│        a) Parsear SKU:                                       │
│           "4J-D-LAPINS-C5WFTFG" →                           │
│             Calibre: "4J"                                    │
│             Calidad: "D"                                     │
│             Variedad: "LAPINS"                               │
│        b) Obtener líneas con getSalidasForSKU():            │
│           - Buscar en assignments del sorter                 │
│           - Normalizar (MAYÚSCULAS) para match               │
│           - Formatear: "L2 L7"                              │
│        c) Escribir fila CSV (semicolon delimiter):           │
│           timestamp;sorter_id;sku;calibre;calidad;           │
│           variedad;lineas;porcentaje;total_skus_activos      │
│    Modo: APPEND (no borra datos anteriores)                 │
│    Output: training_data/training_data.csv                   │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 7. PERSISTIR DATOS ADICIONALES                               │
│    - dataset.json: agregar snapshot al array de snapshots    │
│      (historial completo, crece indefinidamente)             │
│    - current_snapshot.json: sobrescribir con último estado   │
│      (siempre muestra el estado actual)                      │
│    - last_assignments.json: actualizar para próxima comparación│
│    - snapshots_YYYYMMDD.json: backup diario automático       │
│      (se crea uno nuevo cada día)                            │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ 8. MOSTRAR ESTADÍSTICAS (displayStats)                       │
│    Terminal output:                                          │
│      ╔════════════════════════════════════════════════════╗ │
│      ║ Monitoreo #47 - 2025-11-27 17:08:02               ║ │
│      ║ Total snapshots: 47                                ║ │
│      ║ Tiempo activo: 23m 30s                            ║ │
│      ╠════════════════════════════════════════════════════╣ │
│      ║ SORTER 1                                           ║ │
│      ║   4J-D-LAPINS: 26%   L2 L7                        ║ │
│      ║   3J-D-LAPINS: 33%   L5                           ║ │
│      ║   2J-D-LAPINS: 18%   L3 L6                        ║ │
│      ║ SORTER 2                                           ║ │
│      ║   4J-D-LAPINS: 25%   L4                           ║ │
│      ║   3J-D-LAPINS: 35%   L1 L2                        ║ │
│      ╚════════════════════════════════════════════════════╝ │
│    Próxima verificación en 30s...                           │
└────────────────────────┬─────────────────────────────────────┘
                         ▼
                   [SLEEP 30s]
                         │
                         └─────> REPETIR DESDE PASO 1
```

### Función Principal: `Run()`

**Archivo:** `pkg/monitor/monitoreo.go`  
**Responsabilidad:** Orquestador completo del sistema

```go
func Run() {
    // 1. Inicialización
    loadConfig()                         // Cargar config.yaml
    loadExistingDataset()                // Cargar dataset.json si existe
    chartScraper = scraper.New()         // Inicializar chromedp
    startTime := time.Now()
    checkCount := 0
    
    // 2. Loop infinito
    for {
        checkCount++
        timestamp := time.Now()
        
        // 3. Recolección de datos
        assignments := fetchAssignments()  // HTTP GET al API
        snapshot := createSnapshot(timestamp, assignments)
        
        // 4. Detección de cambios
        if hasChanges(lastAssignments, assignments) {
            changes := detectChanges(lastAssignments, assignments)
            logChanges(changes)
            showChangeAlert(changes)
        }
        
        // 5. Persistencia
        dataset.Snapshots = append(dataset.Snapshots, snapshot)
        saveDataset()                     // dataset.json
        saveCurrentSnapshot(snapshot)     // current_snapshot.json
        exportToCSV(snapshot)             // training_data.csv
        
        // 6. Display
        displayStats(checkCount, startTime, snapshot)
        
        // 7. Espera
        time.Sleep(checkInterval)  // 30s (configurable en YAML)
    }
}
```

### Función: `createSnapshot()`

**Algoritmo completo:**
```
INPUT: timestamp, assignments[]
OUTPUT: DataSnapshot completo

1. Inicializar snapshot base:
   - Timestamp formatado
   - DateTime objeto
   - Assignments copiados
   - Contadores por sorter y salida

2. SI captureCharts == true:
   a) chartDataList = chartScraper.ScrapeBothSorters()
   b) snapshot.ChartData = map[sorter_id] → ChartData
   
   c) Calcular calibre_percent (GLOBAL):
      - Inicializar resultado vacío
      - Agregar Sorter 1 percentages
      - SI hay Sorter 2:
        - Para cada SKU en Sorter 2:
          SI SKU ya existe en resultado:
            resultado[SKU] = (s1_percent + s2_percent) / 2.0
          SI NO:
            resultado[SKU] = s2_percent
      - snapshot.CalibrePercent = resultado
   
   d) Para cada sorter con chart_data:
      - Inicializar map vacío para calibre_by_sorter[sorterID]
      - Para cada (sku, percent) en chart.Percentages:
        SI percent > 0:
          calibre_by_sorter[sorterID][sku] = CalibreDistribution{
            Percentage: percent
          }
   
   e) Mapear a SALIDAS:
      - Para cada assignment en sorter específico:
        SI assignment.SKU existe en chart.Percentages:
          realPercent = chart.Percentages[assignment.SKU]
          salida = assignment.Salida
          
          SI calibre_by_salida[salida] no existe:
            calibre_by_salida[salida] = map vacío
          
          calibre_by_salida[salida][sku] = CalibreDistribution{
            Percentage: realPercent
          }
   
   f) Mapear a SORTER+SALIDA:
      - key = fmt.Sprintf("%d-%d", sorterID, salida)
      - Similar al paso (e) pero con key combinada

3. RETURN snapshot completo
```

### Función: `exportToCSV()`

**Formato del CSV generado:**
```csv
timestamp;sorter_id;sku;calibre;calidad;variedad;lineas;porcentaje;total_skus_activos
2025-11-27 17:08:02;1;4J-D-LAPINS-C5WFTFG;4J;D;LAPINS;L2 L7;26.0;11
2025-11-27 17:08:02;1;3J-D-LAPINS-C5WFTFG;3J;D;LAPINS;L5;33.0;11
```

**Características:**
- **Delimitador:** Punto y coma (`;`) para Excel español
- **Encoding:** UTF-8
- **Modo:** APPEND (no borra datos anteriores)
- **Crecimiento:** ~15-20 filas por ciclo → ~1,800 filas/hora

**Propósito:** Dataset limpio para entrenar modelos ML que detecten calibres desde imágenes

---

## 🤖 Sistema de Advisor (IA)

### Arquitectura del Advisor

El sistema de advisor implementa **dos estrategias complementarias** de inteligencia artificial:

```
┌────────────────────────────────────────────────────────┐
│                  ADVISOR SYSTEM                        │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ESTRATEGIA 1: PROMPTING (advisor.go)                 │
│  ┌──────────────────────────────────────────────────┐ │
│  │  LLM (Ollama) con prompt engineering             │ │
│  │  - Modelo: phi3:mini o llama3.2:3b               │ │
│  │  - Endpoint: localhost:11434                     │ │
│  │  - Input: Estado actual (DataSnapshot)           │ │
│  │  - Output: Advice JSON estructurado              │ │
│  │                                                   │ │
│  │  Ventaja: Funciona desde el día 1               │ │
│  │  Uso: Asesoramiento en tiempo real               │ │
│  └──────────────────────────────────────────────────┘ │
│                                                        │
│  ESTRATEGIA 2: FINE-TUNING (decision_inference.go)    │
│  ┌──────────────────────────────────────────────────┐ │
│  │  Inferencia automática + XGBoost                │ │
│  │  - Input: changes_log.json + dataset.json       │ │
│  │  - Proceso: Detecta movimientos automáticamente  │ │
│  │  - Output: decisiones_training.csv               │ │
│  │  - Entrenamiento: train_model.py                 │ │
│  │                                                   │ │
│  │  Ventaja: Aprende patrones históricos            │ │
│  │  Uso: Predicción de razones de decisión          │ │
│  └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
```

### Estrategia 1: Prompting con LLM

**Archivo:** `pkg/advisor/advisor.go`

#### Función Principal: `GetAdvice()`

```go
func GetAdvice(snapshot DataSnapshot) (*Advice, error)
```

**Entrada:** DataSnapshot con estado actual (assignments + chart_data)  
**Salida:** Advice struct con recomendación estructurada

**Proceso:**
```
1. Construir contexto de sorters:
   Para Sorter 1:
     - 4J-D-LAPINS: 26% (líneas L2 L7)
     - 3J-D-LAPINS: 33% (líneas L5)
   Para Sorter 2:
     - 4J-D-LAPINS: 25% (líneas L4)
     - 3J-D-LAPINS: 35% (líneas L1 L2)

2. Generar prompt con template "Don Sergio":
   "Eres Don Sergio, el mejor jefe de packing de Chile con 30 años...
    Reglas sagradas:
    - Nunca dejes diferencia mayor a 8% entre sorters
    - Prioriza mover el SKU con más desbalance
    - Siempre mueve a las líneas del SKU con menos %
    ..."

3. Llamar a Ollama API:
   POST http://localhost:11434/api/generate
   {
     "model": "phi3:mini",
     "prompt": fullPrompt,
     "stream": false,
     "options": {
       "temperature": 0.3,
       "top_p": 0.9
     }
   }

4. Parsear respuesta JSON:
   {
     "accion": "mover" | "no_hacer_nada",
     "sku": "2J-D-LAPINS-C5WFTFG",
     "de_sorter": 1,
     "a_sorter": 2,
     "lineas_sugeridas": "L2 L4 L6",
     "razon": "Explicación corta",
     "balance_esperado_despues": "2J-D ≈ 23% en ambos"
   }

5. Validar y retornar Advice
```

#### Tipo `Advice`

```go
type Advice struct {
    Accion          string  // "mover" o "no_hacer_nada"
    SKU             string  // SKU a mover
    DeSorter        int     // Sorter origen
    ASorter         int     // Sorter destino
    LineasSugeridas string  // "L2 L4 L6"
    Razon           string  // Explicación
    BalanceEsperado string  // Estado esperado después
}
```

#### Tipos Locales (Sin Dependencias Cíclicas)

**Crítico:** `advisor.go` define sus propios tipos para evitar importar `monitor`:

```go
// En pkg/advisor/advisor.go
type Assignment struct {
    Salida   int
    SKU      string
    SorterID int
}

type ChartData struct {
    SorterID    int
    Percentages map[string]float64
    OrderedSKUs []string
}

type DataSnapshot struct {
    Assignments []Assignment
    ChartData   map[int]ChartData
}
```

**¿Por qué?** Si `advisor` importara `monitor`, y `monitor` necesitara importar `advisor`, habría dependencia cíclica. Los tipos locales resuelven esto.

#### Prompt Engineering

**Estrategia del prompt:**
1. **Persona definida:** "Don Sergio" - experiencia y autoridad
2. **Reglas claras:** Nunca >8% diferencia, priorizar desbalances
3. **Output estructurado:** JSON estricto sin texto extra
4. **Contexto completo:** Estado actual con líneas físicas
5. **Temperatura baja:** 0.3 para respuestas más deterministas

**Ejemplo de prompt generado:**
```
Eres Don Sergio, el mejor jefe de packing de Chile con 30 años en sorters de fruta. Hablas directo, claro y con autoridad. Tu misión es balancear los 2 sorters lo más perfecto posible.

Estado actual (porcentajes reales del gráfico):

Sorter 1:
- 4J-D-LAPINS-C5WFTFG: 26% (líneas L2 L7)
- 3J-D-LAPINS-C5WFTFG: 33% (líneas L5)
- 2J-D-LAPINS-C5WFTFG: 18% (líneas L3 L6)

Sorter 2:
- 4J-D-LAPINS-C5WFTFG: 25% (líneas L4)
- 3J-D-LAPINS-C5WFTFG: 35% (líneas L1 L2)
- 2J-D-LAPINS-C5WFTFG: 22% (líneas L5 L7)

Reglas sagradas:
- Nunca dejes diferencia mayor a 8% en la misma variedad entre sorters.
- Prioriza mover el SKU que más desbalance tiene.
- Siempre mueve a las líneas del SKU que tiene menos porcentaje (para no tapar).
- Si está casi perfecto (<6% diferencia en todo), di "no_hacer_nada".

Responde EXACTAMENTE en este JSON, sin texto extra:

{
  "accion": "mover" | "no_hacer_nada",
  "sku": "2J-D-LAPINS-C5WFTFG",
  "de_sorter": 1,
  "a_sorter": 2,
  "lineas_sugeridas": "L2 L4 L6",
  "razon": "Explicación corta y precisa en español",
  "balance_esperado_despues": "2J-D ≈ 23% en ambos sorters"
}
```

**Respuesta esperada del modelo:**
```json
{
  "accion": "mover",
  "sku": "2J-D-LAPINS-C5WFTFG",
  "de_sorter": 2,
  "a_sorter": 1,
  "lineas_sugeridas": "L3 L6",
  "razon": "2J-D está desbalanceado (22% vs 18%). Mover de S2 a S1 para igualar en ~20%.",
  "balance_esperado_despues": "2J-D ≈ 20% en ambos sorters"
}
```

#### Configuración de Ollama

**Instalación:**
```bash
# Windows
winget install Ollama.Ollama

# Verificar instalación
ollama --version
```

**Descargar modelo:**
```bash
# Opción 1: Modelo ligero (recomendado para CPU)
ollama pull phi3:mini

# Opción 2: Modelo más potente (requiere más RAM)
ollama pull llama3.2:3b
```

**Iniciar servidor:**
```bash
ollama serve
# Escucha en http://localhost:11434
```

**Verificar funcionamiento:**
```bash
curl http://localhost:11434/api/generate -d '{
  "model": "phi3:mini",
  "prompt": "Hola",
  "stream": false
}'
```

#### Uso en el Sistema

**Actualmente:** El sistema de prompting está **implementado pero no integrado** en el loop principal.

**Para integrarlo en `cmd/monitor/main.go`:**
```go
import "danich/pkg/advisor"

// Después de createSnapshot()
if checkCount % 10 == 0 {  // Consultar cada 10 ciclos (5 minutos)
    advice, err := advisor.GetAdvice(advisor.DataSnapshot{
        Assignments: snapshot.Assignments,
        ChartData:   convertChartData(snapshot.ChartData),
    })
    
    if err != nil {
        log.Printf("Error obteniendo advice: %v", err)
    } else if advice.Accion == "mover" {
        fmt.Println("\n🤖 RECOMENDACIÓN DE DON SERGIO:")
        fmt.Printf("   Mover %s de Sorter %d a Sorter %d\n", 
            advice.SKU, advice.DeSorter, advice.ASorter)
        fmt.Printf("   Líneas sugeridas: %s\n", advice.LineasSugeridas)
        fmt.Printf("   Razón: %s\n", advice.Razon)
        fmt.Printf("   Balance esperado: %s\n", advice.BalanceEsperado)
    }
}
```

---

## 🧠 Sistema de Inferencia Automática

### ¿Qué problema resuelve?

**Problema:** Para entrenar un modelo ML necesitas datos etiquetados (X → y). En nuestro caso:
- **X:** Estado de los sorters (porcentajes, distribuciones)
- **y:** Razón de la decisión ("desbalance_severo", "optimizacion_preventiva", etc.)

**Solución tradicional:** Una persona revisa cada cambio y etiqueta manualmente.

**Problema:** Imposible en producción. El sistema genera cambios continuamente y no hay tiempo para etiquetar.

**Solución del sistema:** **Inferencia automática** que analiza cambios históricos y deduce la razón sin intervención humana.

### Arquitectura de Inferencia

```
ENTRADA: changes_log.json + dataset.json
    ↓
┌─────────────────────────────────────────┐
│  InferDecisionsFromChanges()            │
│  - Lee todos los cambios históricos     │
│  - Busca snapshot antes/después         │
│  - Analiza movimientos de SKUs          │
└─────────────┬───────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  analyzeChange()                        │
│  - Detecta: ¿Qué SKU se movió?         │
│  - ¿De qué sorter a qué sorter?         │
│  - ¿Qué líneas se agregaron/quitaron?  │
└─────────────┬───────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  inferReason()                          │
│  - Calcula diferencia ANTES             │
│  - Calcula diferencia DESPUÉS           │
│  - Clasifica en 6 categorías:           │
│    1. desbalance_severo (>8%)          │
│    2. desbalance_moderado (5-8%)       │
│    3. sobrecarga_sorter (>80%)         │
│    4. optimizacion_preventiva (<5%)     │
│    5. redistribucion_calibre           │
│    6. ajuste_operacional               │
└─────────────┬───────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  calculateConfidence()                  │
│  - Score 0-1 basado en:                 │
│    • Datos completos (±0.3)            │
│    • Impacto en balance (±0.4)         │
│    • Lógica clara (±0.3)               │
└─────────────┬───────────────────────────┘
              ↓
SALIDA: []InferredDecision
    ↓
┌─────────────────────────────────────────┐
│  decisiones_inferidas.json (detalle)    │
│  decisiones_training.csv (para ML)      │
└─────────────────────────────────────────┘
```

### Archivo: `pkg/advisor/decision_inference.go`

#### Tipo `InferredDecision`

```go
type InferredDecision struct {
    Timestamp string
    SKU       string
    Calibre   string
    Variedad  string
    
    // Movimiento
    DeSorter int
    ASorter  int
    DeLineas string  // "L2 L7"
    ALineas  string  // "L3 L5"
    
    // Estado ANTES del cambio
    PorcentajeAntes_S1 float64
    PorcentajeAntes_S2 float64
    DiferenciaAntes    float64
    TotalSKUsAntes_S1  int
    TotalSKUsAntes_S2  int
    
    // Estado DESPUÉS del cambio
    PorcentajeDespues_S1 float64
    PorcentajeDespues_S2 float64
    DiferenciaDespues    float64
    
    // Inferencia
    RazonInferida  string   // Una de las 6 categorías
    MejoraBalance  bool     // true si diferencia disminuyó
    ImpactoBalance float64  // Reducción de diferencia (positivo = mejora)
    Confianza      float64  // 0-1 (qué tan segura es la inferencia)
}
```

#### Función Principal: `InferDecisionsFromChanges()`

```go
func InferDecisionsFromChanges(changesPath, snapshotsPath string) ([]InferredDecision, error)
```

**Algoritmo completo:**
```
INPUT: 
  - changes_log.json (cambios detectados)
  - dataset.json (snapshots históricos)

OUTPUT:
  - []InferredDecision (decisiones inferidas)

1. Cargar changes_log.json → []ChangeEvent
2. Cargar dataset.json → []SnapshotData

3. Para cada change en changes:
   
   a) Parsear timestamp del cambio
   
   b) Buscar snapshot ANTES:
      - Recorrer snapshots de atrás hacia adelante
      - Encontrar el snapshot cuyo timestamp sea <= changeTime
      - Si no hay, skip este cambio
   
   c) Buscar snapshot DESPUÉS:
      - Recorrer snapshots desde el cambio hacia adelante
      - Encontrar el snapshot cuyo timestamp sea >= changeTime
      - Si no hay, skip este cambio
   
   d) Validar que ambos snapshots tengan chart_data:
      - Si falta data de gráficos, skip (no hay porcentajes)
   
   e) Analizar cambio: movimiento := analyzeChange(snapshotAntes, snapshotDespues, change)
      - Detectar qué SKU se movió
      - De qué sorter a qué sorter
      - Qué líneas cambiaron
   
   f) Si se detectó movimiento válido:
      - razon := inferReason(movimiento, snapshotAntes, snapshotDespues)
      - confianza := calculateConfidence(movimiento, razon)
      - Crear InferredDecision con todos los datos
      - Agregar a lista de decisiones

4. RETURN decisiones[]
```

#### Función: `analyzeChange()`

**Propósito:** Detectar qué SKU se movió y hacia dónde

```
INPUT:
  - snapshotAntes
  - snapshotDespues
  - change (Added, Removed, Modified)

OUTPUT:
  - Movimiento detectado o nil

ALGORITMO:

1. Crear mapa de assignments ANTES:
   mapAntes[key] = Assignment
   key = fmt.Sprintf("%d-%s", sorterID, SKU)

2. Crear mapa de assignments DESPUÉS:
   mapDespues[key] = Assignment

3. Buscar cambios significativos:
   
   a) Para cada REMOVED:
      - key_removed = fmt.Sprintf("%d-%s", removed.SorterID, removed.SKU)
      - Buscar si ese SKU aparece en ADDED en otro sorter
      - SI aparece:
        DETECTADO: Movimiento de SKU de sorter X a sorter Y
        Guardar: DeSorter, ASorter, SKU
   
   b) Si no se detectó movimiento claro:
      - Analizar MODIFIED para ver cambios de líneas
      - Si hay cambio de sorter en modified → movimiento

4. Obtener porcentajes ANTES y DESPUÉS:
   - PorcentajeAntes_S1 = snapshotAntes.ChartData["1"].Percentages[SKU]
   - PorcentajeAntes_S2 = snapshotAntes.ChartData["2"].Percentages[SKU]
   - Similar para DESPUÉS

5. Obtener líneas ANTES y DESPUÉS:
   - DeLineas = buscar assignments del SKU en DeSorter en snapshotAntes
   - ALineas = buscar assignments del SKU en ASorter en snapshotDespues
   - Formatear como "L2 L7"

6. RETURN estructura con todos los datos del movimiento
```

**Ejemplo de detección:**
```
Change:
  Removed: [{"salida": 2, "sku": "4J-D-LAPINS", "sorter_id": 1}]
  Added:   [{"salida": 5, "sku": "4J-D-LAPINS", "sorter_id": 2}]

Detección:
  SKU "4J-D-LAPINS" se movió de Sorter 1 a Sorter 2
  Líneas: De "L2" a "L5"
```

#### Función: `inferReason()`

**Propósito:** Clasificar el tipo de decisión en una de 6 categorías

```go
func inferReason(movimiento Movimiento, snapshotAntes, snapshotDespues SnapshotData) string
```

**Categorías y lógica:**

```
INPUT: Movimiento con porcentajes antes/después

1. Calcular diferencia ANTES:
   diffAntes = |PorcentajeAntes_S1 - PorcentajeAntes_S2|

2. Calcular diferencia DESPUÉS:
   diffDespues = |PorcentajeDespues_S1 - PorcentajeDespues_S2|

3. Calcular carga total de cada sorter:
   cargaAntes_S1 = suma de todos los porcentajes en Sorter 1
   cargaAntes_S2 = suma de todos los porcentajes en Sorter 2

4. CLASIFICAR:

   SI diffAntes > 8.0:
     RETURN "desbalance_severo"
     Razón: Diferencia crítica entre sorters

   SI diffAntes > 5.0 Y diffAntes <= 8.0:
     RETURN "desbalance_moderado"
     Razón: Diferencia moderada que necesita ajuste

   SI cargaAntes_S1 > 80.0 O cargaAntes_S2 > 80.0:
     RETURN "sobrecarga_sorter"
     Razón: Un sorter está sobrecargado (>80%)

   SI diffAntes < 5.0 Y (diffDespues < diffAntes):
     RETURN "optimizacion_preventiva"
     Razón: Mejora proactiva aunque diferencia era pequeña

   SI movimiento involucra cambio de calibre:
     RETURN "redistribucion_calibre"
     Razón: Rebalanceo de tipos de fruta

   SI NO:
     RETURN "ajuste_operacional"
     Razón: Cambio por otras razones operativas
```

**Distribución típica de categorías:**
```
desbalance_severo        : 15%
desbalance_moderado      : 35%
sobrecarga_sorter        : 5%
optimizacion_preventiva  : 10%
redistribucion_calibre   : 20%
ajuste_operacional       : 15%
```

#### Función: `calculateConfidence()`

**Propósito:** Asignar un score de confianza (0-1) a la inferencia

```go
func calculateConfidence(movimiento Movimiento, razon string) float64
```

**Algoritmo:**
```
INICIALIZAR: confidence = 0.5 (base)

1. Verificar completitud de datos (+0.3 max):
   SI tiene porcentajes antes Y después:
     confidence += 0.3
   SI NO:
     confidence += 0.0
     // Inferencia débil sin datos completos

2. Evaluar impacto en balance (+0.4 max):
   impacto = |diffAntes - diffDespues|
   
   SI impacto >= 3.0:  // Cambio significativo
     confidence += 0.4
   SI impacto >= 1.0 Y impacto < 3.0:
     confidence += 0.2
   SI impacto < 1.0:   // Cambio pequeño
     confidence += 0.1

3. Bonus por razones claras (+0.3):
   SI razon == "desbalance_severo" O "desbalance_moderado":
     confidence += 0.3  // Razón muy clara
   SI razon == "sobrecarga_sorter":
     confidence += 0.2  // Razón clara
   SI NO:
     confidence += 0.1  // Razón inferida

4. Asegurar rango [0, 1]:
   SI confidence > 1.0:
     confidence = 1.0

5. RETURN confidence
```

**Ejemplos de confianza:**
```
Caso 1: Desbalance severo con datos completos
  - Datos completos: +0.3
  - Impacto >3%: +0.4
  - Razón clara: +0.3
  → Confianza: 1.0 (máxima)

Caso 2: Optimización preventiva con impacto pequeño
  - Datos completos: +0.3
  - Impacto 1-3%: +0.2
  - Razón inferida: +0.1
  → Confianza: 0.6 (moderada)

Caso 3: Cambio sin porcentajes después
  - Datos incompletos: +0.0
  - Impacto desconocido: +0.1
  - Razón inferida: +0.1
  → Confianza: 0.2 (baja)
```

### Comando: `infer-decisions`

**Archivo:** `cmd/infer-decisions/main.go`

**Propósito:** Ejecutable para generar el dataset de decisiones inferidas

**Uso:**
```bash
./bin/infer-decisions.exe
```

**Proceso:**
```
1. Inicialización:
   - Definir rutas:
     changesPath = "training_data/changes_log.json"
     snapshotsPath = "training_data/dataset.json"
     outputJSON = "training_data/decisiones_inferidas.json"
     outputCSV = "training_data/decisiones_training.csv"

2. Inferencia:
   decisions, err := advisor.InferDecisionsFromChanges(changesPath, snapshotsPath)
   SI error:
     log.Fatal(err)

3. Guardar JSON (detallado):
   jsonData, _ := json.MarshalIndent(decisions, "", "  ")
   ioutil.WriteFile(outputJSON, jsonData, 0644)

4. Guardar CSV (para ML):
   file, _ := os.Create(outputCSV)
   writer := csv.NewWriter(file)
   writer.Comma = ';'  // Semicolon para Excel español
   
   // Headers
   writer.Write([]string{
     "timestamp", "sku", "calibre", "variedad",
     "de_sorter", "a_sorter", "de_lineas", "a_lineas",
     "porcentaje_antes_s1", "porcentaje_antes_s2", "diferencia_antes",
     "porcentaje_despues_s1", "porcentaje_despues_s2", "diferencia_despues",
     "mejora_balance", "impacto_balance",
     "total_skus_antes_s1", "total_skus_antes_s2",
     "razon_inferida", "confianza"
   })
   
   // Datos
   Para cada decision en decisions:
     writer.Write([...]string{
       decision.Timestamp,
       decision.SKU,
       decision.Calibre,
       ...
       decision.RazonInferida,
       fmt.Sprintf("%.2f", decision.Confianza)
     })
   
   writer.Flush()

5. Estadísticas:
   - Total decisiones generadas
   - Distribución por razón
   - Distribución por confianza
   - % que mejoraron balance
   - Impacto promedio en balance
```

**Output ejemplo:**
```
═══════════════════════════════════════════════════════════
     INFERENCIA AUTOMÁTICA DE DECISIONES
═══════════════════════════════════════════════════════════

Analizando cambios históricos...
✓ Cargados 12 cambios de changes_log.json
✓ Cargados 156 snapshots de dataset.json

Procesando...
  Change 1: Detectado movimiento de 4J-D-LAPINS
  Change 2: No se detectó movimiento claro (skip)
  Change 3: Detectado movimiento de 3J-D-LAPINS
  ...

═══════════════════════════════════════════════════════════
     RESULTADOS
═══════════════════════════════════════════════════════════

Total decisiones inferidas: 28

Distribución por razón:
  desbalance_moderado      : 12 (43%)
  redistribucion_calibre   : 8 (29%)
  optimizacion_preventiva  : 4 (14%)
  desbalance_severo        : 2 (7%)
  ajuste_operacional       : 2 (7%)

Distribución por confianza:
  Alta (>0.7)     : 8 (29%)
  Media (0.5-0.7) : 14 (50%)
  Baja (<0.5)     : 6 (21%)

Balance:
  Mejoraron balance: 7 (25%)
  Mantuvieron: 15 (54%)
  Empeoraron: 6 (21%)

Confianza promedio: 0.42
Impacto promedio: 2.1%

═══════════════════════════════════════════════════════════

✓ Datos guardados en:
  - training_data/decisiones_inferidas.json (detalle completo)
  - training_data/decisiones_training.csv (para ML)

Listo para entrenar modelo con train_model.py
```

---

## 🧩 Componentes Principales

### 1. Monitor (`pkg/monitor/monitoreo.go`)

**Responsabilidad:** Orquestar el proceso completo de recolección y análisis de datos

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

## 🎓 Machine Learning y Entrenamiento

### Visión General del Pipeline ML

```
┌─────────────────────────────────────────────────────────┐
│  FASE 1: RECOLECCIÓN (Go - monitor)                    │
│  ────────────────────────────────────────────────       │
│  Loop 30s → dataset.json + changes_log.json + CSV      │
│  Duración: 1-2 semanas                                  │
│  Objetivo: >100 cambios detectados                      │
└────────────┬────────────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────────────┐
│  FASE 2: INFERENCIA (Go - infer-decisions)             │
│  ────────────────────────────────────────────────       │
│  Analiza cambios históricos automáticamente             │
│  Output: decisiones_training.csv con etiquetas          │
│  Objetivo: Dataset etiquetado sin intervención humana   │
└────────────┬────────────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────────────┐
│  FASE 3: ENTRENAMIENTO (Python - train_model.py)       │
│  ────────────────────────────────────────────────       │
│  XGBoost Classifier con 18 features                     │
│  Output: decision_model.pkl + metrics                   │
│  Objetivo: Predecir razón de decisión con >80% accuracy│
└────────────┬────────────────────────────────────────────┘
             ↓
┌─────────────────────────────────────────────────────────┐
│  FASE 4: PREDICCIÓN (Python - usar modelo)             │
│  ────────────────────────────────────────────────       │
│  Cargar modelo entrenado                                │
│  Predecir razón de nuevos cambios                       │
│  Explicar por qué se tomó una decisión                  │
└─────────────────────────────────────────────────────────┘
```

### Dataset de Entrenamiento

**Archivo:** `training_data/decisiones_training.csv`

**Estructura:**
```csv
timestamp;sku;calibre;variedad;de_sorter;a_sorter;de_lineas;a_lineas;porcentaje_antes_s1;porcentaje_antes_s2;diferencia_antes;porcentaje_despues_s1;porcentaje_despues_s2;diferencia_despues;mejora_balance;impacto_balance;total_skus_antes_s1;total_skus_antes_s2;razon_inferida;confianza
```

**Columnas (20 total):**

| Columna | Tipo | Descripción | Ejemplo |
|---------|------|-------------|---------|
| `timestamp` | string | Momento del cambio | "2025-11-27 17:08:02" |
| `sku` | string | SKU completo | "4J-D-LAPINS-C5WFTFG" |
| `calibre` | string | Calibre extraído | "4J" |
| `variedad` | string | Variedad de fruta | "LAPINS" |
| `de_sorter` | int | Sorter origen | 1 |
| `a_sorter` | int | Sorter destino | 2 |
| `de_lineas` | string | Líneas origen | "L2 L7" |
| `a_lineas` | string | Líneas destino | "L3 L5" |
| `porcentaje_antes_s1` | float | % Sorter 1 antes | 26.0 |
| `porcentaje_antes_s2` | float | % Sorter 2 antes | 30.0 |
| `diferencia_antes` | float | Diferencia antes | 4.0 |
| `porcentaje_despues_s1` | float | % Sorter 1 después | 28.0 |
| `porcentaje_despues_s2` | float | % Sorter 2 después | 28.0 |
| `diferencia_despues` | float | Diferencia después | 0.0 |
| `mejora_balance` | bool | ¿Mejoró balance? | true |
| `impacto_balance` | float | Reducción diferencia | 4.0 |
| `total_skus_antes_s1` | int | SKUs activos S1 | 11 |
| `total_skus_antes_s2` | int | SKUs activos S2 | 9 |
| `razon_inferida` | string | **TARGET** Etiqueta | "desbalance_moderado" |
| `confianza` | float | Confianza inferencia | 0.85 |

### Clases de Decisión

El modelo clasifica decisiones en **6 categorías**:

| Clase | Descripción | Criterio | Frecuencia típica |
|-------|-------------|----------|-------------------|
| `desbalance_severo` | Diferencia crítica | diffAntes > 8% | 15% |
| `desbalance_moderado` | Diferencia moderada | 5% < diffAntes ≤ 8% | 35% |
| `sobrecarga_sorter` | Sorter sobrecargado | carga > 80% | 5% |
| `optimizacion_preventiva` | Mejora proactiva | diffAntes < 5% Y mejora | 10% |
| `redistribucion_calibre` | Cambio de calibre | Cambio tipo fruta | 20% |
| `ajuste_operacional` | Otras razones | Resto de casos | 15% |

### Archivo: `train_model.py`

**Propósito:** Script de entrenamiento del modelo XGBoost

**Dependencias:**
```python
pandas==2.3.3
scikit-learn==1.7.2
xgboost==3.1.2
joblib==1.5.2
```

**Instalación:**
```bash
cd MonitoreoDanich
python -m venv venv
source venv/Scripts/activate  # Windows Git Bash
pip install pandas scikit-learn xgboost joblib
```

**Ejecución:**
```bash
python train_model.py
```

### Arquitectura del Modelo

#### 1. Carga y Preparación de Datos

```python
# Cargar CSV con semicolon delimiter
df = pd.read_csv(
    'training_data/decisiones_training.csv',
    sep=';',
    encoding='utf-8'
)

# Validar mínimo de muestras
if len(df) < 50:
    print(f"⚠️  Solo {len(df)} muestras. Recomendado: >100")
    # Continuar de todas formas para testing
```

#### 2. Feature Engineering

**Features Base (9):**
- `de_sorter`, `a_sorter`
- `porcentaje_antes_s1`, `porcentaje_antes_s2`
- `diferencia_antes`
- `porcentaje_despues_s1`, `porcentaje_despues_s2`
- `diferencia_despues`
- `total_skus_antes_s1`, `total_skus_antes_s2`

**Features Derivadas (5):**
```python
# Mejora absoluta en balance
df['mejora_absoluta'] = df['diferencia_antes'] - df['diferencia_despues']

# Mejora relativa (%)
df['mejora_relativa'] = (
    (df['diferencia_antes'] - df['diferencia_despues']) / 
    (df['diferencia_antes'] + 1e-6)
)

# Carga total de cada sorter
df['carga_antes_s1'] = df['porcentaje_antes_s1']  # Simplificado
df['carga_antes_s2'] = df['porcentaje_antes_s2']

# Diferencia de carga entre sorters
df['diff_carga'] = abs(df['carga_antes_s1'] - df['carga_antes_s2'])
```

**Features Categóricas:**
```python
# One-hot encoding de calibre
calibre_dummies = pd.get_dummies(df['calibre'], prefix='calibre')
df = pd.concat([df, calibre_dummies], axis=1)
```

**Total Features:** 18
- 9 base + 5 derivadas + 4 calibre dummies (ej: J, 2J, 3J, 4J)

#### 3. Preparación Target

```python
# Encode target (razon_inferida)
label_encoder = LabelEncoder()
y = label_encoder.fit_transform(df['razon_inferida'])

# Clases:
# 0: ajuste_operacional
# 1: desbalance_moderado
# 2: desbalance_severo
# 3: optimizacion_preventiva
# 4: redistribucion_calibre
# 5: sobrecarga_sorter
```

#### 4. Train/Test Split

```python
X_train, X_test, y_train, y_test = train_test_split(
    X, y,
    test_size=0.2,
    random_state=42
    # NO stratify si hay clases con <2 muestras
)
```

**Problema conocido:** Con pocos datos (<50), algunas clases pueden tener solo 1 muestra, haciendo imposible stratify.

**Solución:** Remover stratify y entrenar con distribución natural.

#### 5. Modelo XGBoost

```python
model = XGBClassifier(
    n_estimators=100,        # Número de árboles
    max_depth=5,             # Profundidad máxima
    learning_rate=0.1,       # Tasa de aprendizaje
    subsample=0.8,           # Fracción de muestras
    colsample_bytree=0.8,    # Fracción de features
    random_state=42,
    eval_metric='mlogloss'   # Métrica de evaluación
)

model.fit(X_train, y_train)
```

**¿Por qué XGBoost?**
- ✅ Excelente con features tabulares
- ✅ Maneja datos desbalanceados
- ✅ Feature importance nativo
- ✅ Robusto con pocos datos
- ✅ No requiere escalado de features

#### 6. Evaluación

```python
# Predicciones
y_pred = model.predict(X_test)

# Accuracy
accuracy = accuracy_score(y_test, y_pred)
print(f"Accuracy: {accuracy:.2f}")

# Reporte detallado
print(classification_report(
    y_test, 
    y_pred,
    target_names=label_encoder.classes_
))

# Feature Importance
importances = model.feature_importances_
for idx in np.argsort(importances)[::-1][:10]:
    print(f"{feature_names[idx]}: {importances[idx]:.3f}")
```

**Features más importantes (típico):**
1. `mejora_absoluta`: 0.27
2. `impacto_balance`: 0.20
3. `diferencia_antes`: 0.15
4. `porcentaje_antes_s1`: 0.12
5. `diferencia_despues`: 0.10

#### 7. Persistencia

```python
# Guardar modelo
joblib.dump(model, 'decision_model.pkl')

# Guardar label encoder
joblib.dump(label_encoder, 'label_encoder.pkl')

# Guardar métricas
metrics = {
    'accuracy': float(accuracy),
    'classification_report': classification_report(
        y_test, y_pred,
        target_names=label_encoder.classes_,
        output_dict=True
    ),
    'feature_importance': dict(zip(feature_names, importances.tolist())),
    'training_samples': len(X_train),
    'test_samples': len(X_test),
    'classes': label_encoder.classes_.tolist()
}

with open('training_metrics.json', 'w') as f:
    json.dump(metrics, f, indent=2)
```

### Output del Entrenamiento

**Ejemplo con datos insuficientes (28 muestras):**
```
═══════════════════════════════════════════════════════════
     ENTRENAMIENTO DE MODELO DE DECISIONES
═══════════════════════════════════════════════════════════

Dataset: training_data/decisiones_training.csv

Cargando datos...
✓ Total decisiones: 28

⚠️  ADVERTENCIA: Solo 28 muestras disponibles
    Se recomienda al menos 100 muestras para entrenamiento robusto
    Continuando de todas formas...

Distribución de clases:
  desbalance_moderado      : 12 (43%)
  redistribucion_calibre   : 8 (29%)
  optimizacion_preventiva  : 4 (14%)
  desbalance_severo        : 2 (7%)
  ajuste_operacional       : 2 (7%)
  sobrecarga_sorter        : 0 (0%)  ⚠️  Clase sin muestras

Features generadas: 18
  - Base: 9
  - Derivadas: 5
  - Calibre (one-hot): 4

Split: 80% train (22), 20% test (6)

Entrenando XGBoost...
✓ Modelo entrenado

═══════════════════════════════════════════════════════════
     RESULTADOS
═══════════════════════════════════════════════════════════

Accuracy (test): 0.50 (3/6)

Classification Report:
                          precision  recall  f1-score  support
desbalance_moderado          0.67     0.67     0.67       3
redistribucion_calibre       0.00     0.00     0.00       2
optimizacion_preventiva      0.00     0.00     0.00       1
desbalance_severo            0.00     0.00     0.00       0
ajuste_operacional           0.00     0.00     0.00       0

           accuracy                          0.50       6
          macro avg          0.13     0.13     0.13      6
       weighted avg          0.45     0.50     0.45      6

Top 10 Features:
  1. mejora_absoluta        : 0.270
  2. impacto_balance        : 0.201
  3. diferencia_antes       : 0.145
  4. porcentaje_antes_s1    : 0.118
  5. diferencia_despues     : 0.095
  6. mejora_relativa        : 0.067
  7. calibre_4J             : 0.053
  8. de_sorter              : 0.031
  9. carga_antes_s1         : 0.020
 10. total_skus_antes_s1    : 0.000

═══════════════════════════════════════════════════════════

✓ Modelo guardado: decision_model.pkl
✓ Label encoder: label_encoder.pkl
✓ Métricas: training_metrics.json

⚠️  RECOMENDACIÓN:
   El modelo tiene baja accuracy (50%) debido a pocas muestras.
   Ejecuta el monitor por 1-2 semanas más para obtener >100 cambios,
   luego vuelve a entrenar para mejor performance.
```

**Ejemplo con datos suficientes (>200 muestras):**
```
═══════════════════════════════════════════════════════════
     ENTRENAMIENTO DE MODELO DE DECISIONES
═══════════════════════════════════════════════════════════

Dataset: training_data/decisiones_training.csv

Cargando datos...
✓ Total decisiones: 247

Distribución de clases:
  desbalance_moderado      : 102 (41%)
  redistribucion_calibre   : 68 (28%)
  optimizacion_preventiva  : 32 (13%)
  ajuste_operacional       : 25 (10%)
  desbalance_severo        : 15 (6%)
  sobrecarga_sorter        : 5 (2%)

Features generadas: 18
Split: 80% train (197), 20% test (50)

Entrenando XGBoost...
✓ Modelo entrenado

═══════════════════════════════════════════════════════════
     RESULTADOS
═══════════════════════════════════════════════════════════

Accuracy (test): 0.86 (43/50)  ✓ Excelente

Classification Report:
                          precision  recall  f1-score  support
desbalance_moderado          0.90     0.95     0.92      20
redistribucion_calibre       0.85     0.79     0.82      14
optimizacion_preventiva      0.75     0.86     0.80       7
ajuste_operacional           0.80     0.67     0.73       6
desbalance_severo            1.00     0.67     0.80       3
sobrecarga_sorter            0.00     0.00     0.00       0

           accuracy                          0.86      50
          macro avg          0.72     0.66     0.68      50
       weighted avg          0.86     0.86     0.86      50

Top 10 Features:
  1. diferencia_antes       : 0.312  ⭐
  2. mejora_absoluta        : 0.285  ⭐
  3. impacto_balance        : 0.168
  4. porcentaje_antes_s1    : 0.092
  5. diferencia_despues     : 0.075
  6. carga_antes_s1         : 0.041
  7. calibre_4J             : 0.027

✓ Modelo guardado: decision_model.pkl
✓ Label encoder: label_encoder.pkl
✓ Métricas: training_metrics.json

🎉 ¡Modelo entrenado exitosamente con 86% accuracy!
   Listo para usar en producción.
```

### Uso del Modelo Entrenado

**Script ejemplo (`predict_decision.py`):**
```python
import joblib
import pandas as pd

# Cargar modelo y encoder
model = joblib.load('decision_model.pkl')
label_encoder = joblib.load('label_encoder.pkl')

# Datos de un nuevo cambio
new_data = pd.DataFrame([{
    'de_sorter': 1,
    'a_sorter': 2,
    'porcentaje_antes_s1': 30.0,
    'porcentaje_antes_s2': 22.0,
    'diferencia_antes': 8.0,
    'porcentaje_despues_s1': 26.0,
    'porcentaje_despues_s2': 26.0,
    'diferencia_despues': 0.0,
    'total_skus_antes_s1': 11,
    'total_skus_antes_s2': 9,
    'calibre': '4J',
    # ... agregar features derivadas ...
}])

# Feature engineering (igual que en training)
new_data['mejora_absoluta'] = new_data['diferencia_antes'] - new_data['diferencia_despues']
# ...

# Predicción
prediction = model.predict(new_data)
predicted_class = label_encoder.inverse_transform(prediction)[0]
probability = model.predict_proba(new_data)[0]

print(f"Razón predicha: {predicted_class}")
print(f"Confianza: {probability.max():.2f}")
print(f"Probabilidades:")
for cls, prob in zip(label_encoder.classes_, probability):
    print(f"  {cls}: {prob:.3f}")
```

**Output:**
```
Razón predicha: desbalance_severo
Confianza: 0.92
Probabilidades:
  desbalance_severo: 0.920
  desbalance_moderado: 0.055
  sobrecarga_sorter: 0.015
  optimizacion_preventiva: 0.008
  redistribucion_calibre: 0.002
  ajuste_operacional: 0.000
```

### Estrategia de Mejora del Modelo

#### Fase 1: Datos insuficientes (0-50 muestras)
- ✅ Usar **Ollama prompting** para asesoramiento
- ⏳ Dejar monitor corriendo 1-2 semanas
- ⏳ NO entrenar modelo aún

#### Fase 2: Datos mínimos (50-100 muestras)
- ✅ Entrenar modelo básico (accuracy ~60-70%)
- ✅ Usar para análisis exploratorio
- ⏳ Seguir recolectando datos

#### Fase 3: Datos suficientes (100-200 muestras)
- ✅ Entrenar modelo robusto (accuracy ~75-85%)
- ✅ Usar en producción con precaución
- ✅ Monitorear predicciones

#### Fase 4: Datos abundantes (>200 muestras)
- ✅ Modelo confiable (accuracy >85%)
- ✅ Producción completa
- ✅ Tuning de hiperparámetros

### Integración del Modelo en el Sistema

**Futuro:** Integrar predicciones en tiempo real

```go
// En pkg/advisor/predictor.go (futuro)
func PredictDecisionReason(change ChangeData) (string, float64, error) {
    // Llamar a Python script desde Go
    cmd := exec.Command("python", "predict_decision.py", 
        "--change", jsonEncode(change))
    
    output, err := cmd.Output()
    if err != nil {
        return "", 0, err
    }
    
    // Parsear resultado
    var result struct {
        Reason     string  `json:"reason"`
        Confidence float64 `json:"confidence"`
    }
    json.Unmarshal(output, &result)
    
    return result.Reason, result.Confidence, nil
}
```

**Uso en monitor:**
```go
// Después de detectChanges()
if hasChanges() {
    reason, confidence, _ := advisor.PredictDecisionReason(changeData)
    
    fmt.Printf("🤖 Predicción ML:\n")
    fmt.Printf("   Razón: %s\n", reason)
    fmt.Printf("   Confianza: %.0f%%\n", confidence*100)
}
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

## 🛠️ Comandos y Herramientas

### Comandos Go

#### 1. `monitor` - Monitoreo Continuo

**Archivo:** `cmd/monitor/main.go`  
**Compilar:**
```bash
go build -o bin/monitor.exe cmd/monitor/main.go
```

**Ejecutar:**
```bash
./bin/monitor.exe
```

**Función:** Loop infinito cada 30s que:
- Captura assignments del API
- Scrapea gráficos con chromedp
- Detecta cambios
- Exporta a CSV y JSON
- Muestra estadísticas en consola

**Output continuo:**
```
✓ Configuración cargada: Danich Cerezas (cereza) - 2 sorters, 7 líneas

════════════════════════════════════════════════════════════
 Monitoreo #1 - 2025-11-27 17:08:02
 Total snapshots: 1
 Tiempo activo: 0m 22s
════════════════════════════════════════════════════════════

SORTER 1 (11 SKUs activos):
  4J-D-LAPINS-C5WFTFG: 26%   L2 L7
  3J-D-LAPINS-C5WFTFG: 33%   L5
  ...

SORTER 2 (9 SKUs activos):
  4J-D-LAPINS-C5WFTFG: 25%   L4
  3J-D-LAPINS-C5WFTFG: 35%   L1 L2
  ...

✓ Sin cambios en assignments
Próxima verificación en 30s...
```

**Detener:** `Ctrl+C`

---

#### 2. `capture-charts` - Testing de Scraper

**Archivo:** `cmd/capture-charts/main.go`  
**Compilar:**
```bash
go build -o bin/capture-charts.exe cmd/capture-charts/main.go
```

**Ejecutar:**
```bash
./bin/capture-charts.exe
```

**Función:** Ejecución única (no loop) para probar el scraper:
- Captura datos de ambos sorters
- Muestra en consola
- Guarda en `chart_data_captured.json`
- Termina

**Uso:** Testing y debugging del scraper sin ejecutar todo el monitor.

---

#### 3. `infer-decisions` - Generación de Dataset ML

**Archivo:** `cmd/infer-decisions/main.go`  
**Compilar:**
```bash
go build -o bin/infer-decisions.exe cmd/infer-decisions/main.go
```

**Ejecutar:**
```bash
./bin/infer-decisions.exe
```

**Función:** Inferencia automática de decisiones:
- Lee `changes_log.json` y `dataset.json`
- Detecta movimientos de SKUs
- Infiere razones automáticamente (6 categorías)
- Calcula confianza (0-1)
- Genera `decisiones_inferidas.json` (detalle completo)
- Genera `decisiones_training.csv` (para ML)
- Muestra estadísticas

**Output:**
```
═══════════════════════════════════════════════════════════
     INFERENCIA AUTOMÁTICA DE DECISIONES
═══════════════════════════════════════════════════════════

Analizando cambios históricos...
✓ Cargados 12 cambios de changes_log.json
✓ Cargados 156 snapshots de dataset.json

Procesando...
  Change 1: Detectado movimiento de 4J-D-LAPINS
  Change 2: No se detectó movimiento claro (skip)
  ...

Total decisiones inferidas: 28

Distribución por razón:
  desbalance_moderado      : 12 (43%)
  redistribucion_calibre   : 8 (29%)
  ...

Confianza promedio: 0.42

✓ Datos guardados en:
  - training_data/decisiones_inferidas.json
  - training_data/decisiones_training.csv
```

**Cuándo ejecutar:**
- Después de 1-2 semanas de monitoreo continuo
- Cuando tengas >10 cambios detectados
- Antes de entrenar el modelo ML

---

#### 4. `analyze-data` - Análisis de Calidad

**Archivo:** `cmd/analyze-data/main.go`  
**Status:** Implementado pero no usado actualmente  
**Compilar:**
```bash
go build -o bin/analyze-data.exe cmd/analyze-data/main.go
```

**Función:** Análisis de calidad y cobertura de datos (futuro).

---

### Comandos Python

#### 1. `train_model.py` - Entrenamiento ML

**Archivo:** `train_model.py`  
**Requisitos:**
```bash
pip install pandas scikit-learn xgboost joblib
```

**Ejecutar:**
```bash
python train_model.py
```

**Función:**
- Carga `decisiones_training.csv`
- Feature engineering (18 features)
- Entrena XGBoost Classifier
- Evalúa con test set
- Guarda modelo, encoder y métricas

**Pre-requisito:** Haber ejecutado `infer-decisions` antes.

**Output:**
```
═══════════════════════════════════════════════════════════
     ENTRENAMIENTO DE MODELO DE DECISIONES
═══════════════════════════════════════════════════════════

Dataset: training_data/decisiones_training.csv
Total decisiones: 247

Distribución de clases: ...
Features generadas: 18
Split: 197 train / 50 test

Entrenando XGBoost...
✓ Modelo entrenado

Accuracy (test): 0.86

✓ Modelo guardado: decision_model.pkl
✓ Label encoder: label_encoder.pkl
✓ Métricas: training_metrics.json
```

---

### Configuración del Entorno Python

#### Setup Inicial

```bash
# 1. Crear entorno virtual
python -m venv venv

# 2. Activar entorno
# Windows (Git Bash)
source venv/Scripts/activate

# Windows (CMD)
venv\Scripts\activate.bat

# Windows (PowerShell)
venv\Scripts\Activate.ps1

# 3. Instalar dependencias
pip install pandas scikit-learn xgboost joblib

# 4. Verificar instalación
python -c "import pandas, sklearn, xgboost, joblib; print('OK')"
```

#### Activación Rápida (cada sesión)

```bash
source venv/Scripts/activate
python train_model.py
```

---

### Workflow Completo

#### Workflow 1: Primera Ejecución (Setup)

```bash
# 1. Compilar monitor (punto de entrada único)
go build -o bin/monitor.exe cmd/monitor/main.go

# Opcional: Compilar herramientas adicionales
go build -o bin/capture-charts.exe cmd/capture-charts/main.go
go build -o bin/infer-decisions.exe cmd/infer-decisions/main.go

# 2. Probar scraper (opcional)
./bin/capture-charts.exe

# 3. Configurar Python
python -m venv venv
source venv/Scripts/activate
pip install pandas scikit-learn xgboost joblib

# 4. Iniciar monitoreo continuo
./bin/monitor.exe
# Dejar corriendo 1-2 semanas...
```

#### Workflow 2: Generar Dataset ML (después de 1-2 semanas)

```bash
# 1. Detener monitor (Ctrl+C)

# 2. Generar decisiones inferidas
./bin/infer-decisions.exe
# Output: decisiones_training.csv

# 3. Entrenar modelo
source venv/Scripts/activate
python train_model.py
# Output: decision_model.pkl

# 4. Reiniciar monitor
./bin/monitor.exe
```

#### Workflow 3: Re-entrenar con más datos (después de 1 mes)

```bash
# 1. Generar nuevo dataset
./bin/infer-decisions.exe
# Ahora tiene más cambios (ej: 200+)

# 2. Re-entrenar modelo
python train_model.py
# Mejor accuracy (ej: 86%)

# 3. Comparar métricas
cat training_metrics.json
```

---

## 🚀 Compilación y Despliegue

### Compilar todos los comandos

```bash
# Monitor principal
go build -o bin/monitor.exe cmd/monitor/main.go

# Herramientas
go build -o bin/capture-charts.exe cmd/capture-charts/main.go
go build -o bin/infer-decisions.exe cmd/infer-decisions/main.go
go build -o bin/analyze-data.exe cmd/analyze-data/main.go

# Compilación cross-platform (Linux desde Windows)
GOOS=linux GOARCH=amd64 go build -o bin/monitor cmd/monitor/main.go
```

### Desplegar en otro packing

```bash
# 1. Copiar binarios
cp bin/*.exe /path/to/otro/packing/

# 2. Editar config.yaml
nano config.yaml
# Cambiar: URL, sorters, líneas, fruta

# 3. Ejecutar
./monitor.exe
```

**No requiere recompilar** - Todo configurable en YAML.

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
3. ✅ Estudiar arquitectura modular en `pkg/monitor/`
   - `monitor.go` → Orquestador principal
   - `models.go` → Estructuras de datos
   - `config.go` → Configuración
   - `fetcher.go` → Cliente API
   - `snapshot.go` → Creación de snapshots
   - `changes.go` → Detección de cambios
   - `persistence.go` → Guardado de datos
   - `exporter.go` → Exportación CSV
   - `display.go` → Visualización
4. ✅ Analizar `pkg/scraper/chart_scraper.go` (scraping)
5. ✅ Revisar modelos de datos
6. ✅ Ejecutar y observar outputs
7. ✅ Modificar configuraciones
8. ✅ Agregar features propios


## 📖 Guía de Uso y Workflows

### Escenario 1: Primera Ejecución del Sistema

**Objetivo:** Iniciar monitoreo desde cero

**Pasos:**
```bash
# 1. Clonar/descargar proyecto
cd MonitoreoDanich

# 2. Configurar config.yaml
nano config.yaml
# Verificar: URL, sorters, líneas corresponden al packing

# 3. Compilar monitor (arquitectura modular)
go build -o bin/monitor.exe cmd/monitor/main.go

# Opcional: Compilar herramientas adicionales
go build -o bin/infer-decisions.exe cmd/infer-decisions/main.go
go build -o bin/capture-charts.exe cmd/capture-charts/main.go

# 4. Probar conectividad (opcional)
curl http://192.168.121.2/api/api/assignments_list

# 5. Iniciar monitor
./bin/monitor.exe
```

**Output esperado:**
- Captura cada 30 segundos
- Muestra distribución de sorters
- Exporta a CSV continuamente
- Detecta y registra cambios

**Dejar corriendo:** 1-2 semanas para obtener datos suficientes (>100 cambios).

---

### Escenario 2: Usar Advisor con Ollama (Prompting)

**Objetivo:** Obtener recomendaciones en tiempo real sin datos históricos

**Pre-requisitos:**
```bash
# 1. Instalar Ollama
winget install Ollama.Ollama

# 2. Descargar modelo
ollama pull phi3:mini
# o para más potencia:
ollama pull llama3.2:3b

# 3. Iniciar servidor Ollama
ollama serve
# Escucha en http://localhost:11434
```

**Uso manual desde Go:**
```go
// En un script de testing
import "danich/pkg/advisor"

snapshot := advisor.DataSnapshot{
    Assignments: assignments,
    ChartData:   chartData,
}

advice, err := advisor.GetAdvice(snapshot)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Acción: %s\n", advice.Accion)
fmt.Printf("SKU: %s\n", advice.SKU)
fmt.Printf("De Sorter %d a Sorter %d\n", advice.DeSorter, advice.ASorter)
fmt.Printf("Líneas sugeridas: %s\n", advice.LineasSugeridas)
fmt.Printf("Razón: %s\n", advice.Razon)
fmt.Printf("Balance esperado: %s\n", advice.BalanceEsperado)
```

**Ventaja:** Funciona inmediatamente sin necesidad de datos históricos.

---

### Escenario 3: Entrenar Modelo ML (Fine-tuning)

**Objetivo:** Crear modelo predictivo basado en decisiones históricas

**Pre-requisitos:**
- Monitor corriendo al menos 1-2 semanas
- Al menos 50 cambios detectados (ideal: >100)

**Pasos:**
```bash
# 1. Verificar cambios recolectados
cat training_data/changes_log.json | grep "change_type" | wc -l
# Si <50, esperar más tiempo

# 2. Generar decisiones inferidas
./bin/infer-decisions.exe

# Verifica output:
# - decisiones_inferidas.json (detalle)
# - decisiones_training.csv (para ML)

# 3. Configurar Python
python -m venv venv
source venv/Scripts/activate
pip install pandas scikit-learn xgboost joblib

# 4. Entrenar modelo
python train_model.py

# Verifica output:
# - decision_model.pkl (modelo entrenado)
# - label_encoder.pkl (encoder de clases)
# - training_metrics.json (métricas)
```

**Resultado esperado:**
- Con 50-100 muestras: accuracy ~60-70% (básico)
- Con 100-200 muestras: accuracy ~75-85% (bueno)
- Con >200 muestras: accuracy >85% (excelente)

**Si accuracy es baja (<60%):**
- Seguir recolectando datos
- Volver a entrenar en 1-2 semanas más

---

### Escenario 4: Analizar Calidad de Datos

**Objetivo:** Verificar que los datos recolectados son correctos

**Validar CSV:**
```bash
# 1. Ver últimas líneas del CSV
tail -n 20 training_data/training_data.csv

# 2. Contar registros por sorter
grep ";1;" training_data/training_data.csv | wc -l  # Sorter 1
grep ";2;" training_data/training_data.csv | wc -l  # Sorter 2

# 3. Ver distribución de calibres
cut -d';' -f4 training_data/training_data.csv | sort | uniq -c

# 4. Validar porcentajes (no deben ser 0)
awk -F';' '$8 == 0 {print}' training_data/training_data.csv
# Si hay muchos con porcentaje 0, hay problema con scraping
```

**Validar JSON:**
```bash
# Ver último snapshot
cat training_data/current_snapshot.json | jq '.chart_data'

# Verificar que ambos sorters tienen datos
cat training_data/current_snapshot.json | jq '.chart_data | keys'
# Esperado: ["1", "2"]
```

---

### Escenario 5: Migrar a Otro Packing

**Objetivo:** Usar el sistema en un packing diferente

**Pasos:**
```bash
# 1. Copiar binarios al nuevo servidor
scp bin/monitor.exe user@new-packing:/path/to/app/

# 2. Crear nuevo config.yaml
cat > config.yaml << EOF
packing:
  name: "Packing XYZ"
  url: "http://10.0.0.50"
  sorters: 3
  lineas: 12
  fruta: "arandano"

monitor:
  intervalo_segundos: 30
  capture_charts: true

data:
  folder: "training_data_xyz"
EOF

# 3. Crear carpeta de datos
mkdir training_data_xyz

# 4. Ejecutar monitor
./monitor.exe
```

**Sin recompilar código** - Todo se adapta del YAML.

---

### Escenario 6: Debugging - Captura No Funciona

**Problema:** El monitor corre pero no captura porcentajes

**Diagnóstico:**
```bash
# 1. Probar captura aislada
./bin/capture-charts.exe

# Si falla, verificar:
# - ¿Chrome está instalado?
# - ¿La URL es correcta en config.yaml?
# - ¿El selector CSS cambió en la web?
```

**Verificar selector CSS:**
```bash
# Inspeccionar HTML de la página
curl http://192.168.121.2/assignment/1 > page.html
cat page.html | grep "percentage"

# Si el selector cambió, editar:
# pkg/scraper/chart_scraper.go
# Buscar: div.relative.w-full.flex.justify-between.items-center
# Actualizar con nuevo selector
```

**Verificar JavaScript:**
```go
// En chart_scraper.go, agregar logs
chromedp.Run(ctx,
    chromedp.Navigate(url),
    chromedp.Sleep(3*time.Second),
    chromedp.Evaluate(`console.log("DOM loaded")`, nil),
    // ... resto del código
)
```

---

### Escenario 7: Monitoreo 24/7 en Producción

**Objetivo:** Ejecutar monitor como servicio continuo

**Opción 1: Screen (Linux)**
```bash
screen -S monitor
./bin/monitor
# Presionar Ctrl+A, luego D para detach
# Reconectar: screen -r monitor
```

**Opción 2: systemd (Linux)**
```ini
# /etc/systemd/system/danich-monitor.service
[Unit]
Description=Danich Sorter Monitor
After=network.target

[Service]
Type=simple
User=monitor
WorkingDirectory=/opt/danich
ExecStart=/opt/danich/bin/monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable danich-monitor
sudo systemctl start danich-monitor
sudo systemctl status danich-monitor
```

**Opción 3: Tarea programada (Windows)**
```
1. Abrir Task Scheduler
2. Create Basic Task
   - Name: Danich Monitor
   - Trigger: At startup
   - Action: Start a program
   - Program: C:\danich\bin\monitor.exe
   - Start in: C:\danich
3. Properties > Settings
   - [x] If task fails, restart every 1 minute
```

---

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

## 📝 Resumen Ejecutivo

### ¿Qué hace este sistema?

MonitoreoDanich es una **plataforma completa de inteligencia artificial** para optimización de sorters de fruta que:

1. **Monitorea continuamente** (cada 30s) el estado de sorters
2. **Recolecta porcentajes reales** desde gráficos HTML/JavaScript
3. **Infiere decisiones automáticamente** sin etiquetado manual
4. **Asesora con IA** mediante dos estrategias:
   - **Prompting (Ollama):** Funciona desde el día 1
   - **Fine-tuning (XGBoost):** Aprende de datos históricos

### Tecnologías Clave

- **Go 1.25.4:** Monitoreo, scraping, inferencia
- **Python 3.11:** Machine learning (XGBoost)
- **chromedp:** Scraping de JavaScript
- **Ollama:** LLM local para prompting
- **XGBoost:** Clasificación de decisiones

### Componentes Principales

| Componente | Archivo | Función |
|------------|---------|---------|
| Monitor | `cmd/monitor/main.go` | Loop continuo de recolección |
| Core Modular | `pkg/monitor/*.go` | 9 archivos especializados (modularizado) |
| Advisor | `pkg/advisor/advisor.go` | Prompting con Ollama |
| Inferencia | `pkg/advisor/decision_inference.go` | Etiquetado automático |
| Entrenamiento | `train_model.py` | Fine-tuning XGBoost |

### Datos Generados

| Archivo | Contenido | Uso |
|---------|-----------|-----|
| `dataset.json` | Snapshots históricos | Análisis retrospectivo |
| `changes_log.json` | Cambios detectados | Input para inferencia |
| `training_data.csv` | Porcentajes + features | Dataset base ML |
| `decisiones_inferidas.json` | Decisiones con razón | Detalle completo |
| `decisiones_training.csv` | Dataset etiquetado | Input para XGBoost |
| `decision_model.pkl` | Modelo entrenado | Predicción de razones |

### Estado Actual

**✅ Implementado:**
- ✅ Monitoreo continuo funcionando
- ✅ Scraping de gráficos con chromedp
- ✅ Detección de cambios
- ✅ Exportación a CSV y JSON
- ✅ **Arquitectura modular (9 archivos especializados)**
- ✅ Advisor con Ollama (no integrado en loop)
- ✅ Sistema de inferencia automática
- ✅ Script de entrenamiento ML

**⚠️ En progreso:**
- Recolección de datos (necesita >100 cambios para ML confiable)
- Entrenamiento del modelo (accuracy mejorará con más datos)

**🔜 Próximos pasos:**
1. Dejar monitor corriendo 1-2 semanas más
2. Re-ejecutar `infer-decisions` con más cambios
3. Re-entrenar modelo con >100 muestras
4. Integrar predicciones en monitor
5. Integrar asesoramiento Ollama en loop

### Documentos Relacionados

- **DOCUMENTACION_TECNICA.md** (este archivo): Arquitectura completa y detalles técnicos
- **FLUJO_DATOS.md**: Flujo detallado paso a paso desde API hasta CSV
- **README.md**: Guía rápida de instalación y uso
- **todo.md**: TODOs y mejoras futuras
- **config.yaml**: Configuración activa del sistema

---

**Proyecto:** MonitoreoDanich  
**Versión:** 2.0  
**Autor:** Sistema de Monitoreo y Advisor de Sorters  
**Última actualización:** 27 Noviembre 2025  
**Lenguajes:** Go 1.25.4 + Python 3.11  
**Licencia:** Uso interno - Danich Packing
