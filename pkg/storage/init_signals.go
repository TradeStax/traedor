package storage

import (
	"fmt"
	"log"
)

// InitializeSignalDefinitions ensures all required signal definitions exist in the database
func (p *PostgresStorage) InitializeSignalDefinitions() error {
	signalDefs := []SignalDefinition{
		{
			Name:        "RSI",
			Description: "Relative Strength Index - Momentum oscillator measuring speed and magnitude of price changes",
			Type:        "rsi",
			Parameters: map[string]interface{}{
				"period":           14,
				"overbought_level": 70,
				"oversold_level":   30,
			},
			Active: true,
		},
		{
			Name:        "RSI 5m",
			Description: "RSI on 5-minute aggregated bars",
			Type:        "rsi",
			Parameters: map[string]interface{}{
				"period":              14,
				"overbought_level":    70,
				"oversold_level":      30,
				"aggregation_interval": 5,
			},
			Active: true,
		},
		{
			Name:        "RSI 30m",
			Description: "RSI on 30-minute aggregated bars",
			Type:        "rsi",
			Parameters: map[string]interface{}{
				"period":              14,
				"overbought_level":    70,
				"oversold_level":      30,
				"aggregation_interval": 30,
			},
			Active: true,
		},
		{
			Name:        "SMA",
			Description: "Simple Moving Average - Calculates average price over specified periods",
			Type:        "sma_crossover",
			Parameters: map[string]interface{}{
				"shortPeriod": 10,
				"longPeriod":  20,
			},
			Active: true,
		},
		{
			Name:        "SMA 5m",
			Description: "SMA crossover on 5-minute aggregated bars",
			Type:        "sma_crossover",
			Parameters: map[string]interface{}{
				"shortPeriod":          10,
				"longPeriod":           20,
				"aggregation_interval": 5,
			},
			Active: true,
		},
		{
			Name:        "SMA 30m",
			Description: "SMA crossover on 30-minute aggregated bars",
			Type:        "sma_crossover",
			Parameters: map[string]interface{}{
				"shortPeriod":          10,
				"longPeriod":           20,
				"aggregation_interval": 30,
			},
			Active: true,
		},
	}

	for _, def := range signalDefs {
		// Check if signal definition already exists
		var count int
		err := p.db.QueryRow("SELECT COUNT(*) FROM signal_definitions WHERE name = $1", def.Name).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existing signal definition %s: %w", def.Name, err)
		}

		if count == 0 {
			// Create the signal definition
			err = p.CreateSignalDefinition(def)
			if err != nil {
				return fmt.Errorf("failed to create signal definition %s: %w", def.Name, err)
			}
			log.Printf("Created signal definition: %s", def.Name)
		} else {
			log.Printf("Signal definition already exists: %s", def.Name)
		}
	}

	return nil
}

// GetSignalDefinitionIDByName retrieves the ID of a signal definition by its name
func (p *PostgresStorage) GetSignalDefinitionIDByName(name string) (string, error) {
	// Check cache first
	if id, ok := p.signalDefCache[name]; ok {
		return id, nil
	}
	
	var id string
	err := p.db.QueryRow("SELECT id FROM signal_definitions WHERE name = $1", name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to get signal definition ID for %s: %w", name, err)
	}
	
	// Update cache
	p.signalDefCache[name] = id
	return id, nil
}

// populateSignalDefCache populates the cache with signal definition name -> ID mappings
func (p *PostgresStorage) populateSignalDefCache() error {
	rows, err := p.db.Query("SELECT id, name FROM signal_definitions")
	if err != nil {
		return fmt.Errorf("failed to query signal definitions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return fmt.Errorf("failed to scan signal definition: %w", err)
		}
		p.signalDefCache[name] = id
	}

	return rows.Err()
}