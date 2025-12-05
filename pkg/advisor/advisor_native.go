 package advisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// AdvisorConfig configuración del advisor
type AdvisorConfig struct {
	OllamaURL   string
	OllamaModel string
	Timeout     time.Duration
}

// SorterData datos de un sorter específico
type SorterData struct {
	SKUs map[string]SKUInfo `json:"skus"`
}

// SKUInfo información de un SKU
type SKUInfo struct {
	Percentage float64 `json:"percentage"`
	Lines      []int   `json:"lines"`
}

// SystemState estado completo del sistema
type SystemState struct {
	Timestamp time.Time  `json:"timestamp"`
	Sorter1   SorterData `json:"sorter_1"`
	Sorter2   SorterData `json:"sorter_2"`
}

// Advice recomendación del advisor
type Advice struct {
	Accion    string `json:"accion"`
	SKU       string `json:"sku,omitempty"`
	DeSorter  int    `json:"de_sorter,omitempty"`
	ASorter   int    `json:"a_sorter,omitempty"`
	Razon     string `json:"razon"`
	Timestamp string `json:"timestamp"`
}

// Imbalance representa un desbalance detectado
type Imbalance struct {
	SKU        string
	Sorter1Pct float64
	Sorter2Pct float64
	Difference float64
	Priority   float64
}

// Advisor implementa la lógica de asesoramiento
type Advisor struct {
	config AdvisorConfig
	client *http.Client
}

// NewAdvisor crea una nueva instancia del advisor
func NewAdvisor(config AdvisorConfig) *Advisor {
	return &Advisor{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetAdvice genera una recomendación basada en el estado actual
func (a *Advisor) GetAdvice(state SystemState) (*Advice, error) {
	fmt.Println("🔍 Analizando estado del sistema...")

	// Detectar desbalances críticos
	imbalances := a.detectImbalances(state)

	if len(imbalances) == 0 {
		return &Advice{
			Accion:    "mantener",
			Razon:     "Sistema balanceado - todas las diferencias <8%",
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// Tomar el desbalance más crítico
	worst := imbalances[0]
	fmt.Printf("📊 Desbalance crítico detectado: %s (%.1f%% diferencia)\n",
		worst.SKU, worst.Difference)

	// Determinar dirección del movimiento
	var deSorter, aSorter int
	if worst.Sorter1Pct > worst.Sorter2Pct {
		deSorter = 1
		aSorter = 2
	} else {
		deSorter = 2
		aSorter = 1
	}

	// Crear advice básico
	advice := &Advice{
		Accion:   "mover",
		SKU:      worst.SKU,
		DeSorter: deSorter,
		ASorter:  aSorter,
		Razon: fmt.Sprintf("Desbalance crítico: %.1f%% diferencia (S%d:%.1f%% vs S%d:%.1f%%)",
			worst.Difference, deSorter,
			worst.Sorter1Pct, aSorter, worst.Sorter2Pct),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Intentar enriquecer con Ollama si está disponible
	if a.config.OllamaModel != "" {
		enhancedAdvice, err := a.enhanceWithOllama(state, advice)
		if err != nil {
			fmt.Printf("⚠️ Ollama no disponible: %v\n", err)
			return advice, nil
		}
		return enhancedAdvice, nil
	}

	return advice, nil
}

// detectImbalances encuentra desbalances críticos entre sorters
func (a *Advisor) detectImbalances(state SystemState) []Imbalance {
	var imbalances []Imbalance

	// Obtener todos los SKUs únicos
	allSKUs := make(map[string]bool)
	for sku := range state.Sorter1.SKUs {
		allSKUs[sku] = true
	}
	for sku := range state.Sorter2.SKUs {
		allSKUs[sku] = true
	}

	// Analizar cada SKU
	for sku := range allSKUs {
		s1Info, s1Exists := state.Sorter1.SKUs[sku]
		s2Info, s2Exists := state.Sorter2.SKUs[sku]

		var s1Pct, s2Pct float64
		if s1Exists {
			s1Pct = s1Info.Percentage
		}
		if s2Exists {
			s2Pct = s2Info.Percentage
		}

		difference := math.Abs(s1Pct - s2Pct)

		// Solo considerar desbalances significativos
		if difference > 8.0 {
			priority := calculatePriority(s1Pct, s2Pct, difference)

			imbalances = append(imbalances, Imbalance{
				SKU:        sku,
				Sorter1Pct: s1Pct,
				Sorter2Pct: s2Pct,
				Difference: difference,
				Priority:   priority,
			})
		}
	}

	// Ordenar por prioridad (mayor primero)
	for i := 0; i < len(imbalances)-1; i++ {
		for j := i + 1; j < len(imbalances); j++ {
			if imbalances[i].Priority < imbalances[j].Priority {
				imbalances[i], imbalances[j] = imbalances[j], imbalances[i]
			}
		}
	}

	return imbalances
}

// calculatePriority calcula la prioridad de un desbalance
func calculatePriority(s1Pct, s2Pct, difference float64) float64 {
	// Factores que aumentan la prioridad:
	// 1. Mayor diferencia absoluta
	// 2. Mayor carga total (s1 + s2)
	// 3. Mayor desproporción relativa

	totalLoad := s1Pct + s2Pct
	relativeImbalance := difference / (totalLoad + 1) // +1 para evitar división por 0

	return difference * (1 + totalLoad/100) * (1 + relativeImbalance)
}

// enhanceWithOllama mejora el advice usando Ollama
func (a *Advisor) enhanceWithOllama(state SystemState, basicAdvice *Advice) (*Advice, error) {
	prompt := a.buildPrompt(state, basicAdvice)

	ollamaResponse, err := a.queryOllama(prompt)
	if err != nil {
		return nil, err
	}

	// Intentar parsear la respuesta JSON de Ollama
	var enhancedAdvice Advice
	err = json.Unmarshal([]byte(ollamaResponse), &enhancedAdvice)
	if err != nil {
		// Si Ollama no devolvió JSON válido, usar advice básico pero con explicación mejorada
		basicAdvice.Razon = fmt.Sprintf("%s. Análisis: %s", basicAdvice.Razon, ollamaResponse)
		return basicAdvice, nil
	}

	// Asegurar timestamp
	enhancedAdvice.Timestamp = time.Now().Format(time.RFC3339)
	return &enhancedAdvice, nil
}

// buildPrompt construye el prompt para Ollama
func (a *Advisor) buildPrompt(state SystemState, advice *Advice) string {
	var prompt bytes.Buffer

	prompt.WriteString("ESTADO ACTUAL DEL SISTEMA:\n\n")

	prompt.WriteString("Sorter 1:\n")
	for sku, info := range state.Sorter1.SKUs {
		if info.Percentage > 0 {
			prompt.WriteString(fmt.Sprintf("  %s: %.1f%% (líneas %v)\n",
				sku, info.Percentage, info.Lines))
		}
	}

	prompt.WriteString("\nSorter 2:\n")
	for sku, info := range state.Sorter2.SKUs {
		if info.Percentage > 0 {
			prompt.WriteString(fmt.Sprintf("  %s: %.1f%% (líneas %v)\n",
				sku, info.Percentage, info.Lines))
		}
	}

	prompt.WriteString("\nDESBALANCES DETECTADOS:\n")
	imbalances := a.detectImbalances(state)
	for _, imb := range imbalances {
		if len(imbalances) <= 3 { // Solo mostrar los top 3
			prompt.WriteString(fmt.Sprintf("  %s: %.1f%% diferencia (S1:%.1f%% vs S2:%.1f%%)\n",
				imb.SKU, imb.Difference, imb.Sorter1Pct, imb.Sorter2Pct))
		}
	}

	prompt.WriteString(fmt.Sprintf("\nSUGERENCIA INICIAL: %s\n", advice.Razon))
	prompt.WriteString("\nPor favor confirma o mejora esta sugerencia siguiendo las reglas de balanceo.")

	return prompt.String()
}

// queryOllama envía una consulta a Ollama
func (a *Advisor) queryOllama(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":       a.config.OllamaModel,
		"prompt":      prompt,
		"stream":      false,
		"temperature": 0.1,
		"options": map[string]interface{}{
			"num_predict": 200,
			"top_p":       0.9,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		a.config.OllamaURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&ollamaResp)
	if err != nil {
		return "", err
	}

	response, ok := ollamaResp["response"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format from Ollama")
	}

	return response, nil
}

// DefaultConfig retorna una configuración por defecto
func DefaultConfig() AdvisorConfig {
	return AdvisorConfig{
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "danich-advisor", // Modelo fine-tuneado
		Timeout:     15 * time.Second,
	}
}
