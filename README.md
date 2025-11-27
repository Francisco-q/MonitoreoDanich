# Sistema de Monitoreo de Calibres - Danich

Sistema de recolección de datos en tiempo real para entrenar modelos de Machine Learning que detecten porcentajes de calibres de fruta desde gráficos de barras en sorters industriales.

## 📋 Índice

1. [¿Qué hace este proyecto?](#qué-hace-este-proyecto)
2. [Estructura del proyecto](#estructura-del-proyecto)
3. [Requisitos previos](#requisitos-previos)
4. [Instalación](#instalación)
5. [Uso](#uso)
6. [Datos generados](#datos-generados)
7. [Arquitectura](#arquitectura)
8. [Glosario](#glosario)

---

## 🎯 ¿Qué hace este proyecto?

Este sistema **monitorea continuamente** los sorters de fruta y **captura datos reales** de los gráficos HTML para:

1. **Capturar porcentajes reales** de cada SKU desde los gráficos visuales (no calculados)
2. **Monitorear assignments** (asignaciones de SKUs a salidas)
3. **Detectar cambios** en las configuraciones
4. **Generar dataset** para entrenar modelos de ML

### Ejemplo de dato capturado:

```json
{
  "timestamp": "2025-11-27 11:47:50",
  "sorter_1": {
    "4J-D-SANTINA-C5WFTFG": 26,
    "3J-D-SANTINA-C5WFTFG": 33,
    "2J-D-SANTINA-C5WFTFG": 13,
    "J-D-SANTINA-C5CZMG": 1,
    "J-D-SANTINA-C5WFTFG": 5,
    "XL-D-SANTINA-C5CZGR": 0
  }
}
```

---

## 📁 Estructura del proyecto

```
Danich/
├── cmd/                          # Programas ejecutables
│   ├── monitor/                  # Monitor continuo (PRINCIPAL)
│   │   └── main.go              # Loop cada 30 segundos
│   └── capture-charts/          # Captura única (TESTING)
│       └── main.go              # Ejecuta una sola vez
│
├── pkg/                          # Código compartido
│   ├── monitor/
│   │   └── monitoreo.go         # Lógica del monitor
│   └── scraper/
│       ├── scraper.go           # Scraper de API
│       └── chart_scraper.go     # Scraper de gráficos HTML
│
├── training_data/                # DATOS GENERADOS ⭐
│   ├── dataset.json             # Dataset completo
│   ├── current_snapshot.json    # Última captura
│   ├── changes_log.json         # Historial de cambios
│   └── snapshots_YYYYMMDD.json  # Respaldos diarios
│
├── bin/                          # Ejecutables compilados
│   ├── monitor.exe              # Monitor principal
│   └── capture-charts.exe       # Herramienta de testing
│
├── go.mod                        # Dependencias de Go
└── README.md                     # Esta guía
```

---

## ✅ Requisitos previos

- **Go 1.21+** instalado ([Descargar](https://go.dev/dl/))
- **Acceso a la red** donde están los sorters (IP: `192.168.121.2`)
- **Google Chrome** instalado (para chromedp)

### Verificar instalación:

```bash
go version
# Debe mostrar: go version go1.21.x windows/amd64
```

---

## 🚀 Instalación

### 1. Clonar o descargar el proyecto

```bash
cd "C:\Users\Francisco\Desktop\Danich"
```

### 2. Descargar dependencias

```bash
go mod download
```

### 3. Compilar los programas

```bash
# Compilar monitor principal
go build -o bin/monitor.exe cmd/monitor/main.go

# Compilar herramienta de testing
go build -o bin/capture-charts.exe cmd/capture-charts/main.go
```

---

## 💻 Uso

### Opción 1: Monitor continuo (RECOMENDADO)

**Propósito:** Recolectar datos continuamente para el dataset de ML

```bash
./bin/monitor.exe
```

**¿Qué hace?**
- ✅ Verifica assignments cada 30 segundos
- ✅ Captura porcentajes reales de gráficos HTML
- ✅ Detecta cambios automáticamente
- ✅ Guarda snapshots en `training_data/dataset.json`
- ✅ Genera logs de cambios

**Salida esperada:**

```
=== Monitor de Asignaciones - Recolección de Datos ===
URL: http://192.168.121.2/api/api/assignments_list
Intervalo de verificación: 30s
Captura de gráficos: true
============================================================
✓ Scraper de gráficos inicializado
✓ Dataset cargado: 10 snapshots desde 2025-11-27 11:36:21

[2025-11-27 11:47:50] Verificación #1
✓ Obtenidos 21 assignments
📊 Gráficos capturados: 2 sorters con porcentajes reales

------------------------------------------------------------
📊 Estadísticas de recolección:
  • Total snapshots: 10
  • Assignments actuales: 21
  • Por sorter: Sorter 1=10, Sorter 2=11
  • Por salida: S1=2 S2=2 S3=3 S4=3 S5=4 S6=4 S7=3

  • Distribución por Sorter (datos reales del gráfico):
    Sorter 1:
      4J-D-SANTINA-C5WFTFG: 26%
      3J-D-SANTINA-C5CZMG: 0%
      3J-D-SANTINA-C5WFTFG: 33%
      2J-D-SANTINA-C5CZMG: 0%
      2J-D-SANTINA-C5WFTFG: 13%
      2J-L-SANTINA-C5WFTFG: 6%
      J-D-SANTINA-C5CZMG: 1%
      J-D-SANTINA-C5WFTFG: 5%
      XL-D-SANTINA-C5CZGR: 0%
------------------------------------------------------------

Próxima verificación en 30s...
```

**Para detener:** Presiona `Ctrl+C`

---

### Opción 2: Captura única (TESTING)

**Propósito:** Probar el scraper o hacer capturas puntuales

```bash
./bin/capture-charts.exe
```

**¿Qué hace?**
- ✅ Ejecuta una sola vez
- ✅ Captura gráficos de ambos sorters
- ✅ Muestra resultado en consola
- ✅ Guarda en `chart_data_captured.json`

---

## 📊 Datos generados

### 1. `training_data/dataset.json`

**Archivo principal** con todos los snapshots para ML:

```json
{
  "collection_start": "2025-11-27T11:36:21Z",
  "collection_end": "2025-11-27T11:50:00Z",
  "total_snapshots": 15,
  "snapshots": [
    {
      "timestamp": "2025-11-27 11:47:50",
      "total_count": 21,
      "by_sorter": {
        "1": 10,
        "2": 11
      },
      "chart_data": {
        "1": {
          "sorter_id": 1,
          "percentages": {
            "4J-D-SANTINA-C5WFTFG": 26,
            "3J-D-SANTINA-C5WFTFG": 33,
            "2J-D-SANTINA-C5WFTFG": 13
          },
          "ordered_skus": [
            "4J-D-SANTINA-C5WFTFG",
            "3J-D-SANTINA-C5CZMG",
            "3J-D-SANTINA-C5WFTFG"
          ]
        }
      },
      "calibre_by_sorter": {
        "1": {
          "4J-D-SANTINA-C5WFTFG": {
            "percentage": 26
          }
        }
      },
      "calibre_by_salida": {
        "1": {
          "4J-D-SANTINA-C5WFTFG": {
            "percentage": 26
          }
        }
      }
    }
  ]
}
```

### 2. `training_data/current_snapshot.json`

Última captura en tiempo real.

### 3. `training_data/changes_log.json`

Historial de todos los cambios detectados:

```json
[
  {
    "timestamp": "2025-11-27 11:48:30",
    "change_type": "update",
    "added": [...],
    "removed": [...],
    "modified": [...],
    "description": "Agregados: 2, Eliminados: 1, Modificados: 3"
  }
]
```

---

## 🏗️ Arquitectura

### Patrón: Hexagonal + Event-Driven

```
┌─────────────────────────────────────────┐
│         MONITOR (Loop 30s)              │
│  - Obtiene assignments del API          │
│  - Captura gráficos con chromedp        │
│  - Detecta cambios                      │
│  - Genera snapshots                     │
└─────────────────────────────────────────┘
                    ↓
        ┌───────────────────────┐
        │   ChartScraper        │
        │  (chromedp)           │
        │  - Abre browser       │
        │  - Ejecuta JavaScript │
        │  - Extrae porcentajes │
        └───────────────────────┘
                    ↓
        ┌───────────────────────┐
        │   Datos Reales        │
        │  SKU → Porcentaje     │
        │  (del gráfico HTML)   │
        └───────────────────────┘
                    ↓
        ┌───────────────────────┐
        │   Dataset JSON        │
        │  - Snapshots          │
        │  - Distribuciones     │
        │  - Historial          │
        └───────────────────────┘
```

### Flujo de datos:

1. **API** → assignments (SKU, sorter_id, salida)
2. **Chromedp** → gráficos HTML (SKU → porcentaje real)
3. **Monitor** → combina ambos datos
4. **Dataset** → guarda para ML

---

## 📖 Glosario

### Conceptos del dominio:

- **Sorter**: Máquina clasificadora de fruta (hay 2: Sorter 1 y Sorter 2)
- **Salida**: Canal de salida del sorter (del 1 al 7)
- **SKU**: Código del producto (formato: `CALIBRE-CALIDAD-VARIEDAD-LOTE`)
  - Ejemplo: `4J-D-SANTINA-C5WFTFG`
  - `4J` = Cuádruple Jumbo (calibre)
  - `D` = Calidad
  - `SANTINA` = Variedad de cereza
  - `C5WFTFG` = Código de lote

- **Calibre**: Tamaño de la fruta
  - `XL` = Extra Large
  - `J` = Jumbo
  - `2J` = Doble Jumbo
  - `3J` = Triple Jumbo
  - `4J` = Cuádruple Jumbo

- **Assignment**: Asignación de un SKU a una salida específica

### Conceptos técnicos:

- **Snapshot**: Captura de estado en un momento específico
- **Dataset**: Conjunto de snapshots para entrenar ML
- **Chromedp**: Librería para controlar Chrome headless (navegador sin interfaz)
- **Scraping**: Extracción automatizada de datos desde HTML

---

## 🔧 Configuración

### Modificar intervalo de monitoreo:

Editar `pkg/monitor/monitoreo.go`:

```go
const (
    checkInterval = 30 * time.Second  // Cambiar a 60 para 1 minuto
)
```

### Cambiar URL del API:

Editar `pkg/monitor/monitoreo.go`:

```go
const (
    baseURL = "http://192.168.121.2"  // Tu IP
)
```

### Activar/desactivar captura de gráficos:

```go
const (
    captureCharts = true  // false para solo capturar assignments
)
```

---

## 🐛 Solución de problemas

### Error: "go: command not found"

**Solución:** Agregar Go al PATH
```bash
export PATH=$PATH:/c/Program\ Files/Go/bin
```

### Error: "chromedp: chrome failed to start"

**Solución:** Instalar Google Chrome

### Error: "connection refused"

**Solución:** Verificar que estás en la red correcta y el API está disponible:
```bash
curl http://192.168.121.2/api/api/assignments_list
```

### Monitor no detecta cambios

**Causa:** Los assignments no han cambiado
**Solución:** Espera a que haya cambios reales en el sorter

---

## 📈 Próximos pasos

1. ✅ **Recolectar datos**: Dejar corriendo el monitor por varios días
2. ⏳ **Análisis de datos**: Explorar el dataset generado
3. ⏳ **Entrenamiento ML**: Usar los datos para entrenar modelo
4. ⏳ **Predicción**: Implementar modelo que detecte calibres desde imágenes

---

## 📝 Notas importantes

- Los **porcentajes son reales** del gráfico HTML (no calculados manualmente)
- El **orden de SKUs** se mantiene igual al gráfico original
- Los **snapshots se acumulan** en un solo archivo para facilitar el análisis
- El sistema **detecta cambios** automáticamente y los registra

---

## 🤝 Contacto

- Proyecto: Recolección de datos para ML - Detección de calibres
- Fecha: Noviembre 2025
- Tecnologías: Go, chromedp, JSON

---

**¡Listo para entrenar tu modelo! 🚀**
