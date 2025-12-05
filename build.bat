@echo off
echo 🔨 Compilando Sistema MonitoreoDanich
echo ====================================

:: Crear directorio bin si no existe
if not exist bin mkdir bin

:: Compilar monitor principal
echo 📦 Compilando monitor principal...
go build -o bin/monitor.exe cmd/monitor/main.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Error compilando monitor
    exit /b 1
)
echo ✅ monitor.exe compilado exitosamente

:: Compilar preparador de Ollama
echo 📦 Compilando prepare-ollama...
go build -o bin/prepare-ollama.exe cmd/prepare-ollama/main.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Error compilando prepare-ollama
    exit /b 1
)
echo ✅ prepare-ollama.exe compilado exitosamente

:: Compilar inferencia de decisiones si existe
if exist "cmd/infer-decisions/main.go" (
    echo 📦 Compilando infer-decisions...
    go build -o bin/infer-decisions.exe cmd/infer-decisions/main.go
    if %ERRORLEVEL% NEQ 0 (
        echo ❌ Error compilando infer-decisions
    ) else (
        echo ✅ infer-decisions.exe compilado exitosamente
    )
)

echo.
echo 🎯 COMPILACIÓN COMPLETADA
echo ========================
echo.
echo Binarios disponibles en bin/:
dir bin\*.exe
echo.
echo Para usar el sistema:
echo 1. bin\prepare-ollama.exe     # Preparar modelo fine-tuneado
echo 2. ollama create danich-advisor -f training_data/Modelfile
echo 3. bin\monitor.exe            # Iniciar monitoreo con advisor integrado