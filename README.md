# Sistema de Monitoreo y Optimización de Producción - Danich

Monitoreo en tiempo real de líneas de clasificación de frutas con análisis inteligente de distribución de carga usando Ollama.

## 🎯 Descripción

Sistema que monitorea dos sorters (líneas de clasificación) independientes, captura datos de producción en tiempo real, detecta cambios, y genera sugerencias para optimizar la distribución de carga **dentro de cada sorter** usando reglas + LLM local (Ollama).

## 🏗️ Arquitectura

```
┌──────────────────────────────────────────────────────────┐
│                MONITOR (Go - Orquestador)                 │
│  • Scraping gráficos (chromedp)                          │
│  • Fetch assignments HTTP                                │
│  • Detección de cambios                                  │
│  • Persistencia JSON/CSV                                 │
└──────────────┬───────────────────────────┬───────────────┘
               │                           │
       ┌───────▼───────┐         ┌────────▼─────────┐
       │ Chart Scraper │         │  API Assignments │
       │  (chromedp)   │         │   (HTTP GET)     │
       └───────────────┘         └──────────────────┘
               │
               └──────────────┬────────────────┐
                              │                │
                   ┌──────────▼──────────┐     │
                   │ Advisor Client (Go) │     │
                   │  HTTP POST          │     │
                   └──────────┬──────────┘     │
                              │                │
                   ┌──────────▼──────────────┐ │
                   │ Flask Server (Python)   │ │
                   │  localhost:5000         │ │
                   └──────────┬──────────────┘ │
                              │                │
                   ┌──────────▼──────────────┐ │
                   │  Hybrid Advisor         │ │
                   │  Análisis de carga      │ │
                   └──────────┬──────────────┘ │
                              │                │
                   ┌──────────▼──────────────┐ │
                   │  Ollama (llama3.2:3b)   │ │
                   │  localhost:11434        │ │
                   └─────────────────────────┘ │
                                               │
                              ┌────────────────▼───────────┐
                              │ Archivos JSON/CSV          │
                              │ training_data/             │
                              └────────────────────────────┘
```

## 📦 Componentes

### Monitor (Go)
- `cmd/monitor/main.go` - Entry point
- `pkg/monitor/` - Lógica de monitoreo (config, fetcher, snapshot, changes, persistence, display)
- `pkg/scraper/chart_scraper.go` - Scraping con chromedp
- `pkg/advisor/advisor_client.go` - Cliente HTTP a Flask

### Advisor (Python)
- `advisor_server.py` - Flask API REST (puerto 5000)
- `hybrid_advisor.py` - Análisis de carga + Ollama

### Monitor ZPL (Python)
- `monitorzpl.py` - Monitor de cajas sin etiqueta ZPL en PostgreSQL

## 🚀 Instalación

### Requisitos
- Go 1.21+
- Python 3.11+
- Ollama + modelo llama3.2:3b
- Chrome/Chromium

### Setup

1. **Ollama**:
```bash
ollama pull llama3.2:3b
ollama serve
```

2. **Python**:
```bash
python -m venv venv
source venv/Scripts/activate
pip install -r requirements.txt
```

3. **Compilar monitor**:
```bash
go build -o bin/monitor.exe cmd/monitor/main.go
```

## 📖 Uso

**Terminal 1** - Ollama:
```bash
ollama serve
```

**Terminal 2** - Advisor:
```bash
python advisor_server.py
```

**Terminal 3** - Monitor:
```bash
./bin/monitor.exe
```

**Terminal 4** (opcional) - Monitor ZPL:
```bash
python monitorzpl.py
```

## 🔧 Configuración

`config.yaml`:
```yaml
packing:
  name: "Frutizano"
  url: "http://192.168.121.2"
  sorters: 2
  lineas: 7

monitor:
  intervalo_segundos: 30
  capture_charts: true

data:
  folder: "training_data"

assignments_url: "http://192.168.121.2/api/api/assignments_list"
```

## 💡 Lógica del Advisor

**Concepto clave**: Los sorters son **procesos paralelos independientes**. No se comparan entre sí, solo se optimiza la distribución **dentro de cada sorter**.

### Análisis
1. Por cada sorter, busca SKUs con >40% de carga
2. Si tiene >2 líneas asignadas, sugiere concentrar en menos líneas
3. Objetivo: liberar líneas para otros SKUs y optimizar capacidad

### Ejemplo
```
Sorter 1:
  SKU: 3J-L-LAPINS → 56.8% en líneas [5, 3]
  
Análisis:
  - Carga alta (>40%)
  - Distribuido en 2 líneas (28.4% c/u)
  - Sugerencia: Concentrar en 1 línea para liberar capacidad
```

## 📊 Output del Monitor

```
[2025-12-01 17:36:38] Verificación #52
✓ Obtenidos 17 assignments
📊 Gráficos capturados: 2 sorters

Distribución por Sorter:
  Sorter 1:
    3J-L-LAPINS-C5REDBTFG: 57% (líneas [5,3])
    2J-D-LAPINS-C5REDBTFG: 25% (líneas [1,2])
  Sorter 2:
    3J-D-LAPINS-C5REDBTFG: 34% (líneas [6])
    2J-L-LAPINS-C5REDBTFG: 47% (líneas [2,5,4])

💡 SUGERENCIA DE OPTIMIZACIÓN
══════════════════════════════════
SKU: 3J-L-LAPINS-C5REDBTFG
Sorter: 1
Líneas actuales: [5 3] (2 líneas)
Líneas sugeridas: [5] (1 línea)

Carga total: 56.8%
Carga por línea (actual): 28.4%
Carga por línea (sugerida): 56.8%

📋 Razón: Concentrar carga: 2 → 1 líneas en S1 para optimizar capacidad
🦙 Ollama: Concentrar la carga libera capacidad para otros SKUs...
══════════════════════════════════
```

## 📁 Datos Persistidos

```
training_data/
├── dataset.json              # Histórico completo de snapshots
├── training_data.csv         # Snapshots en CSV (flat)
├── changes_log.json          # Log de cambios detectados
├── current_snapshot.json     # Estado más reciente
└── flujo_historico.csv       # Datos históricos (6,809 registros)
```

## ⚙️ Funcionalidades

- ✅ Scraping de porcentajes reales desde gráficos HTML
- ✅ Fetch de assignments con líneas asignadas
- ✅ Normalización de SKUs (MAYÚSCULAS) para matching consistente
- ✅ Detección de cambios (agregados, eliminados, modificados)
- ✅ Análisis de carga dentro de cada sorter
- ✅ Sugerencias con explicación en lenguaje natural (Ollama)
- ✅ Exportación automática JSON + CSV
- ✅ Monitor ZPL para PostgreSQL

## 📈 Performance

- **Ciclo completo**: ~20 segundos
  - Fetch assignments: ~1s
  - Scraping (2 sorters): ~12s
  - Análisis + Ollama: ~15s
- **Timeout Ollama**: 25 segundos
- **Intervalo de monitoreo**: 30 segundos

## 🔗 Endpoints

- **Charts**: `http://192.168.121.2/assignment/{1,2}`
- **Assignments**: `http://192.168.121.2/api/api/assignments_list`
- **Advisor**: `http://localhost:5000/analyze`
- **Ollama**: `http://localhost:11434/api/generate`

## 🐛 Troubleshooting

**Advisor no responde**:
```bash
curl http://localhost:5000/health
```

**Ollama timeout**:
```bash
curl http://localhost:11434/api/tags
# Aumentar timeout en advisor_client.go línea 52
```

**SKUs sin líneas**:
- Verificar normalización a MAYÚSCULAS en `monitor.go`
- Revisar formato de response del API assignments

## 👥 Autor

Francisco - Danich
