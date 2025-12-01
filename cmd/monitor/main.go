package main

import (
	"log"

	"danich/pkg/monitor"
)

func main() {
	m, err := monitor.New()
	if err != nil {
		log.Fatalf("Error inicializando monitor: %v", err)
	}

	if err := m.Run(); err != nil {
		log.Fatalf("Error ejecutando monitor: %v", err)
	}
}
