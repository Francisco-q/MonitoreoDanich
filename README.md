# MonitoreoDanich

Sistema de monitoreo continuo para sorters de fruta que captura datos reales de gráficos HTML y genera datasets para entrenamiento de modelos ML.

## Propósito

Recolectar automáticamente porcentajes de calibres desde gráficos de barras en tiempo real para entrenar modelos que detecten calibres automáticamente

---

## ⚙️ Instalación

Requisitos: Go 1.21+ y Google Chrome.

```bash
# Descargar dependencias
go mod download

# Compilar monitor
go build -o bin/monitor.exe cmd/monitor/main.go
```

---

## 🚀 Uso

```bash
./bin/monitor.exe
```

El monitor se ejecuta cada 30 segundos, captura assignments del API y scrapea los porcentajes reales de los gráficos. Los datos se guardan automáticamente en `training_data/`.

---

## 📁 Datos generados

- `training_data/dataset.json` - Snapshots con porcentajes reales
- `training_data/training_data.csv` - CSV para ML con columnas: timestamp, sorter_id, sku, calibre, calidad, variedad, lineas, porcentaje, total_skus_activos
- `training_data/current_snapshot.json` - Última captura
- `training_data/changes_log.json` - Historial de cambios

---

## ⚙️ Configuración

Edita `config.yaml` para ajustar:
- URL del packing y sorters
- Líneas de selladora activas
- Intervalo de monitoreo
- Tipo de fruta

Ver `config.example.yaml` para ejemplos de diferentes packings.

---

## 📖 Documentación

- `DOCUMENTACION_TECNICA.md` - Arquitectura, funciones, escalabilidad completa
- `config.example.yaml` - Ejemplos de configuración

---
