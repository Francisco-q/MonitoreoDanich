package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Fetcher maneja la obtención de datos del API
type Fetcher struct {
	assignmentsURL string
}

// NewFetcher crea un nuevo fetcher
func NewFetcher(assignmentsURL string) *Fetcher {
	return &Fetcher{
		assignmentsURL: assignmentsURL,
	}
}

// FetchAssignments obtiene los assignments actuales del API
func (f *Fetcher) FetchAssignments() ([]Assignment, error) {
	resp, err := http.Get(f.assignmentsURL)
	if err != nil {
		return nil, fmt.Errorf("error en GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("código de estado: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo body: %w", err)
	}

	var assignments []Assignment
	if err := json.Unmarshal(body, &assignments); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %w", err)
	}

	return assignments, nil
}
