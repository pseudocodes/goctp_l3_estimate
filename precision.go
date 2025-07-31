package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
	"unicode"
)

// ExchangeInfo represents Binance exchange info response
type ExchangeInfo struct {
	Symbols []SymbolInfo `json:"symbols"`
}

// SymbolInfo represents information about a trading symbol
type SymbolInfo struct {
	Symbol  string   `json:"symbol"`
	Filters []Filter `json:"filters"`
}

// Filter represents a symbol filter (price, lot size, etc.)
type Filter struct {
	FilterType string `json:"filterType"`
	TickSize   string `json:"tickSize,omitempty"`
	StepSize   string `json:"stepSize,omitempty"`
}

// PrecisionInfo holds precision data for a symbol
type PrecisionInfo struct {
	Symbol         string `json:"symbol"`
	PricePrecision int    `json:"price_precision"`
	QtyPrecision   int    `json:"qty_precision"`
	TickSize       string `json:"tick_size"`
	StepSize       string `json:"step_size"`
	LastUpdated    int64  `json:"last_updated"`
}

// PrecisionManager manages precision information for symbols
type PrecisionManager struct {
	precisions map[string]*PrecisionInfo
	mu         sync.RWMutex
	client     *http.Client
}

// NewPrecisionManager creates a new precision manager
func NewPrecisionManager() *PrecisionManager {
	return &PrecisionManager{
		precisions: make(map[string]*PrecisionInfo),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// calculatePrecision calculates decimal places from a step size string
func calculatePrecision(stepSize string) int {
	if stepSize == "" {
		return 2 // Default precision
	}

	stepFloat, err := strconv.ParseFloat(stepSize, 64)
	if err != nil || stepFloat <= 0 {
		return 2 // Default precision
	}

	if stepFloat >= 1.0 {
		return 0
	}

	// Calculate precision from step size
	precision := int(math.Ceil(-math.Log10(stepFloat)))
	if precision < 0 {
		precision = 0
	}
	if precision > 10 { // Reasonable upper limit
		precision = 10
	}

	return precision
}

func calculatePrecision2(stepFloat float64) int {
	// Calculate precision from step size
	precision := int(math.Ceil(-math.Log10(stepFloat)))
	if precision < 0 {
		precision = 0
	}
	if precision > 10 { // Reasonable upper limit
		precision = 10
	}

	return precision
}

// FetchPrecisionInfo fetches precision information for a symbol from Binance
func (pm *PrecisionManager) FetchPrecisionInfo(symbol string) (*PrecisionInfo, error) {
	pm.mu.RLock()
	if info, exists := pm.precisions[symbol]; exists {
		// Check if info is recent (cache for 1 hour)
		if time.Now().Unix()-info.LastUpdated < 3600 {
			pm.mu.RUnlock()
			return info, nil
		}
	}
	pm.mu.RUnlock()

	ct := ExtractContractPrefix(symbol)
	resp, err := GetInstruments(
		[]string{"futures"},
		[]string{},   // 所有国家/地区
		[]string{},   // 所有交易所
		[]string{ct}, // 所有品种
	)
	// Fetch from API
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instrument info: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("symbol %s not found in exchange info", symbol)
	}

	priceTick := resp.Data[0].PriceTick
	log.Printf("priceTick: %f", priceTick)

	pricePrecision := calculatePrecision2(priceTick)

	var formatStr string
	if pricePrecision >= 1 {
		formatStr = fmt.Sprintf("%%.%df", pricePrecision)
	} else {
		formatStr = "%.0f"
	}
	// log.Printf("formatStr: %s", formatStr)
	tickSize := fmt.Sprintf(formatStr, priceTick)
	log.Printf("pricePrecision: %d, priceTick: %f, tickSize: %s", pricePrecision, priceTick, tickSize)
	// Find the symbol in the response

	precisionInfo := &PrecisionInfo{
		Symbol:         symbol,
		PricePrecision: pricePrecision,
		QtyPrecision:   1,        // Default
		TickSize:       tickSize, // Default
		StepSize:       "1",      // Default
		LastUpdated:    time.Now().Unix(),
	}
	return precisionInfo, nil
}

// GetPrecisionInfo gets cached precision info or fetches it if not available
func (pm *PrecisionManager) GetPrecisionInfo(symbol string) *PrecisionInfo {
	info, err := pm.FetchPrecisionInfo(symbol)
	if err != nil {
		log.Printf("Failed to fetch precision for %s: %v, using defaults", symbol, err)
		return &PrecisionInfo{
			Symbol:         symbol,
			PricePrecision: 1,
			QtyPrecision:   1,
			TickSize:       "1",
			StepSize:       "1",
			LastUpdated:    time.Now().Unix(),
		}
	}
	return info
}

// FormatPrice formats a price with the correct precision for the symbol
func (pm *PrecisionManager) FormatPrice(symbol string, price float64) string {
	info := pm.GetPrecisionInfo(symbol)
	format := fmt.Sprintf("%%.%df", info.PricePrecision)
	return fmt.Sprintf(format, price)
}

// FormatQuantity formats a quantity with the correct precision for the symbol
func (pm *PrecisionManager) FormatQuantity(symbol string, qty float64) string {
	info := pm.GetPrecisionInfo(symbol)
	format := fmt.Sprintf("%%.%df", info.QtyPrecision)
	return fmt.Sprintf(format, qty)
}

// GetAllPrecisionInfo returns all cached precision info
func (pm *PrecisionManager) GetAllPrecisionInfo() map[string]*PrecisionInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*PrecisionInfo)
	for k, v := range pm.precisions {
		result[k] = v
	}
	return result
}

// ClearCache clears the precision cache
func (pm *PrecisionManager) ClearCache() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.precisions = make(map[string]*PrecisionInfo)
}

// Global precision manager instance
var precisionManager *PrecisionManager

// InitializePrecisionManager initializes the global precision manager
func InitializePrecisionManager() {
	precisionManager = NewPrecisionManager()
}

// ExtractContractPrefix 提取合约字符串中前面的非数字字符
// 例如: "rb2508" -> "rb", "TA509" -> "TA"
func ExtractContractPrefix(contract string) string {
	if contract == "" {
		return ""
	}

	var prefix []rune
	for _, char := range contract {
		// 如果遇到数字，停止提取
		if unicode.IsDigit(char) {
			break
		}
		prefix = append(prefix, char)
	}

	return string(prefix)
}
