package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Kopiere die ursprünglichen Einstellungen aus dem Bot
var (
	symbols = []string{"❌", "❓", "🍒", "🍋", "🍊", "🍇", "⭐", "💎", "💰"}
	originalFrequencies = []int{9, 13, 20, 15, 12, 11, 7, 3, 1}
	originalPayoutFactors = map[string]float32{
		// Standard Kombinationen
		"❓❓❓": 2,
		"🍒🍒🍒": 2,
		"🍋🍋🍋": 2.5,
		"🍊🍊🍊": 4,
		"🍇🍇🍇": 10,
		"⭐⭐⭐": 20,
		"💎💎💎": 40,
		"💰💰💰": 500,

		// Erweiterte Kombinationen mit ?
		"🍒🍒❓": 0.5, "🍒❓🍒": 0.5, "❓🍒🍒": 0.5,
		"🍋🍋❓": 0.75, "🍋❓🍋": 0.75, "❓🍋🍋": 0.75,
		"🍊🍊❓": 1.2, "🍊❓🍊": 1.2, "❓🍊🍊": 1.2,
		"🍇🍇❓": 2, "🍇❓🍇": 2, "❓🍇🍇": 2,
		"⭐⭐❓": 3, "⭐❓⭐": 3, "❓⭐⭐": 3,
		"💎💎❓": 5, "💎❓💎": 5, "❓💎💎": 5,
		"💰💰❓": 8, "💰❓💰": 8, "❓💰💰": 8,

		// Erweiterung mit Money Bag
		"💰💰🍒": 10, "💰🍒💰": 10, "🍒💰💰": 10,
		"💰💰🍋": 12, "💰🍋💰": 12, "🍋💰💰": 12,
		"💰💰🍊": 15, "💰🍊💰": 15, "🍊💰💰": 15,
		"💰💰🍇": 20, "💰🍇💰": 20, "🍇💰💰": 20,
		"💰💰⭐": 30, "💰⭐💰": 30, "⭐💰💰": 30,
		"💰💰💎": 80, "💰💎💰": 80, "💎💰💰": 80,

		"❓❓💰": 4, "❓💰❓": 4, "💰❓❓": 4,
		"🍒🍒💰": 5, "🍒💰🍒": 5, "💰🍒🍒": 5,
		"🍋🍋💰": 6, "🍋💰🍋": 6, "💰🍋🍋": 6,
		"🍊🍊💰": 8, "🍊💰🍊": 8, "💰🍊🍊": 8,
		"🍇🍇💰": 14, "🍇💰🍇": 14, "💰🍇🍇": 14,
		"⭐⭐💰": 25, "⭐💰⭐": 25, "💰⭐⭐": 25,
		"💎💎💰": 60, "💎💰💎": 60, "💰💎💎": 60,
	}

	lines = [][][2]int{
		{{0, 0}, {0, 1}, {0, 2}}, // Horizontal oben
		{{1, 0}, {1, 1}, {1, 2}}, // Horizontal Mitte
		{{2, 0}, {2, 1}, {2, 2}}, // Horizontal unten
		{{0, 0}, {1, 0}, {2, 0}}, // Vertikal links
		{{0, 1}, {1, 1}, {2, 1}}, // Vertikal Mitte
		{{0, 2}, {1, 2}, {2, 2}}, // Vertikal rechts
		{{0, 0}, {1, 1}, {2, 2}}, // Diagonal \\
		{{0, 2}, {1, 1}, {2, 0}}, // Diagonal /
	}
)

type SlotConfig struct {
	Frequencies    []int
	PayoutFactors  map[string]float32
	ExpectedRoI    float64
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Runde auf 1 Nachkommastelle
func roundToOneDecimal(value float32) float32 {
	return float32(math.Round(float64(value)*10) / 10)
}

// Runde alle Auszahlungsfaktoren auf 1 Nachkommastelle
func roundPayoutFactors(payouts map[string]float32) map[string]float32 {
	rounded := make(map[string]float32)
	for key, value := range payouts {
		rounded[key] = roundToOneDecimal(value)
	}
	return rounded
}

func getRandomSymbolWithFreq(frequencies []int) string {
	total := 0
	for _, freq := range frequencies {
		total += freq
	}

	rnd := rand.Intn(total)
	cumulative := 0
	for i, freq := range frequencies {
		cumulative += freq
		if rnd < cumulative {
			return symbols[i]
		}
	}
	return symbols[len(symbols)-1]
}

func spinSlotMachine(frequencies []int) [3][3]string {
	var board [3][3]string
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			board[i][j] = getRandomSymbolWithFreq(frequencies)
		}
	}
	return board
}

func calculatePayout(board [3][3]string, bet int, payoutFactors map[string]float32) float32 {
	var payout float32 = 0
	processedLines := make(map[string]bool)

	for _, line := range lines {
		var lineSymbols []string
		var lineKey []string

		for _, pos := range line {
			symbol := board[pos[0]][pos[1]]
			lineSymbols = append(lineSymbols, symbol)
			lineKey = append(lineKey, fmt.Sprintf("%d-%d", pos[0], pos[1]))
		}

		lineKeyStr := fmt.Sprintf("%v", lineKey)
		lineSymbolStr := fmt.Sprintf("%s%s%s", lineSymbols[0], lineSymbols[1], lineSymbols[2])

		if factor, exists := payoutFactors[lineSymbolStr]; exists {
			if !processedLines[lineKeyStr] {
				payout += float32(bet) * factor
				processedLines[lineKeyStr] = true
			}
		}
	}

	return payout
}

func simulateGames(numGames int, frequencies []int, payoutFactors map[string]float32) float64 {
	totalBet := float64(numGames) // Annahme: Einsatz von 1 pro Spiel
	totalPayout := float64(0)

	for i := 0; i < numGames; i++ {
		board := spinSlotMachine(frequencies)
		payout := calculatePayout(board, 1, payoutFactors)
		totalPayout += float64(payout)
	}

	return totalPayout / totalBet
}

func copyMap(original map[string]float32) map[string]float32 {
	copy := make(map[string]float32)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func copySlice(original []int) []int {
	copy := make([]int, len(original))
	for i, v := range original {
		copy[i] = v
	}
	return copy
}

// Mutiere Frequenzen zufällig (kleine Änderungen)
func mutateFrequencies(frequencies []int, intensity float64) []int {
	mutated := copySlice(frequencies)
	for i := range mutated {
		if rand.Float64() < 0.3 { // 30% Chance für Änderung
			change := int(float64(mutated[i]) * intensity * (rand.Float64()*2 - 1)) // -intensity bis +intensity
			mutated[i] = int(math.Max(1, float64(mutated[i]+change))) // Mindestens 1
		}
	}
	return mutated
}

// Mutiere Auszahlungsfaktoren zufällig (kleine Änderungen)
func mutatePayoutFactors(payouts map[string]float32, intensity float64) map[string]float32 {
	mutated := copyMap(payouts)
	for key, value := range mutated {
		if rand.Float64() < 0.2 { // 20% Chance für Änderung
			change := float32(float64(value) * intensity * (rand.Float64()*2 - 1))
			newValue := value + change
			if newValue > 0.1 { // Mindestens 0.1
				mutated[key] = roundToOneDecimal(newValue)
			}
		}
	}
	return mutated
}

func optimizeRoI(targetRoI float64, numSimulations int) SlotConfig {
	fmt.Printf("Starte Optimierung für Ziel-RoI: %.2f%%\n", targetRoI*100)
	
	bestConfig := SlotConfig{
		Frequencies:   copySlice(originalFrequencies),
		PayoutFactors: roundPayoutFactors(copyMap(originalPayoutFactors)),
	}
	
	// Teste aktuelle Konfiguration
	currentRoI := simulateGames(numSimulations/10, bestConfig.Frequencies, bestConfig.PayoutFactors)
	bestConfig.ExpectedRoI = currentRoI
	
	fmt.Printf("Aktuelle RoI: %.4f%% (%.2f%% Abweichung vom Ziel)\n", 
		currentRoI*100, math.Abs(currentRoI-targetRoI)*100)

	bestDiff := math.Abs(currentRoI - targetRoI)
	
	// Strategie 1: Globale Skalierung der Auszahlungsfaktoren
	fmt.Println("\n=== Strategie 1: Globale Skalierung der Auszahlungsfaktoren ===")
	
	scaleFactor := targetRoI / currentRoI
	scaledPayouts := copyMap(originalPayoutFactors)
	
	for key, value := range scaledPayouts {
		scaledPayouts[key] = roundToOneDecimal(value * float32(scaleFactor))
	}
	
	scaledRoI := simulateGames(numSimulations/5, originalFrequencies, scaledPayouts)
	scaledDiff := math.Abs(scaledRoI - targetRoI)
	
	fmt.Printf("Skalierungsfaktor: %.4f\n", scaleFactor)
	fmt.Printf("Resultierende RoI: %.4f%% (%.2f%% Abweichung)\n", 
		scaledRoI*100, scaledDiff*100)
	
	if scaledDiff < bestDiff {
		bestConfig.PayoutFactors = scaledPayouts
		bestConfig.ExpectedRoI = scaledRoI
		bestDiff = scaledDiff
		fmt.Println("✓ Neue beste Konfiguration gefunden!")
	}

	// Strategie 2: Genetischer Algorithmus
	fmt.Println("\n=== Strategie 2: Genetischer Algorithmus (Frequencies + Payouts) ===")
	
	population := 20
	generations := 50
	
	// Erstelle Anfangspopulation
	configs := make([]SlotConfig, population)
	for i := 0; i < population; i++ {
		if i == 0 {
			// Erste Konfiguration ist die beste bisherige
			configs[i] = SlotConfig{
				Frequencies:   copySlice(bestConfig.Frequencies),
				PayoutFactors: copyMap(bestConfig.PayoutFactors),
			}
		} else {
			// Zufällige Variationen der besten Konfiguration
			configs[i] = SlotConfig{
				Frequencies:   mutateFrequencies(bestConfig.Frequencies, 0.1),
				PayoutFactors: roundPayoutFactors(mutatePayoutFactors(bestConfig.PayoutFactors, 0.1)),
			}
		}
	}
	
	for generation := 0; generation < generations; generation++ {
		// Bewerte alle Konfigurationen
		for i := range configs {
			configs[i].ExpectedRoI = simulateGames(10000, configs[i].Frequencies, configs[i].PayoutFactors)
		}
		
		// Finde beste Konfiguration
		bestIdx := 0
		bestGenDiff := math.Abs(configs[0].ExpectedRoI - targetRoI)
		
		for i := 1; i < len(configs); i++ {
			diff := math.Abs(configs[i].ExpectedRoI - targetRoI)
			if diff < bestGenDiff {
				bestIdx = i
				bestGenDiff = diff
			}
		}
		
		// Aktualisiere globale beste Konfiguration
		if bestGenDiff < bestDiff {
			bestConfig = SlotConfig{
				Frequencies:   copySlice(configs[bestIdx].Frequencies),
				PayoutFactors: copyMap(configs[bestIdx].PayoutFactors),
				ExpectedRoI:   configs[bestIdx].ExpectedRoI,
			}
			bestDiff = bestGenDiff
		}
		
		if generation%10 == 0 {
			fmt.Printf("Generation %d: Beste RoI = %.4f%% (%.2f%% Abweichung)\n", 
				generation, configs[bestIdx].ExpectedRoI*100, bestGenDiff*100)
		}
		
		// Erstelle neue Generation (Elitismus + Mutation)
		newConfigs := make([]SlotConfig, population)
		
		// Top 5 übernehmen (Elitismus)
		indices := make([]int, len(configs))
		for i := range indices {
			indices[i] = i
		}
		
		// Sortiere nach Fitness
		for i := 0; i < len(indices)-1; i++ {
			for j := i + 1; j < len(indices); j++ {
				diff1 := math.Abs(configs[indices[i]].ExpectedRoI - targetRoI)
				diff2 := math.Abs(configs[indices[j]].ExpectedRoI - targetRoI)
				if diff2 < diff1 {
					indices[i], indices[j] = indices[j], indices[i]
				}
			}
		}
		
		for i := 0; i < 5; i++ {
			newConfigs[i] = configs[indices[i]]
		}
		
		// Rest durch Mutation der besten 5 erzeugen
		for i := 5; i < population; i++ {
			parent := newConfigs[i%5]
			intensity := 0.05 + rand.Float64()*0.1 // 5-15% Mutation
			
			newConfigs[i] = SlotConfig{
				Frequencies:   mutateFrequencies(parent.Frequencies, intensity),
				PayoutFactors: roundPayoutFactors(mutatePayoutFactors(parent.PayoutFactors, intensity)),
			}
		}
		
		configs = newConfigs
	}
	
	fmt.Printf("Nach genetischem Algorithmus: RoI = %.4f%% (%.2f%% Abweichung)\n", 
		bestConfig.ExpectedRoI*100, bestDiff*100)

	// Strategie 3: Finale Feinabstimmung
	fmt.Println("\n=== Strategie 3: Finale Feinabstimmung ===")
	
	for iteration := 0; iteration < 30; iteration++ {
		testRoI := simulateGames(50000, bestConfig.Frequencies, bestConfig.PayoutFactors)
		diff := testRoI - targetRoI
		
		if math.Abs(diff) < 0.0005 { // Sehr nahe am Ziel
			break
		}
		
		// Kleine Anpassungen der Auszahlungsfaktoren
		adjustmentFactor := 1.0 + (diff * -0.02) // Kleinere Anpassungen
		
		newPayouts := copyMap(bestConfig.PayoutFactors)
		for key, value := range newPayouts {
			newPayouts[key] = roundToOneDecimal(value * float32(adjustmentFactor))
		}
		
		newRoI := simulateGames(20000, bestConfig.Frequencies, newPayouts)
		newDiff := math.Abs(newRoI - targetRoI)
		
		if newDiff < bestDiff {
			bestConfig.PayoutFactors = newPayouts
			bestConfig.ExpectedRoI = newRoI
			bestDiff = newDiff
		}
		
		if iteration%5 == 0 {
			fmt.Printf("Feintuning %d: RoI = %.4f%%, Anpassung = %.4f\n", 
				iteration, testRoI*100, adjustmentFactor)
		}
	}
	
	// Finale Bewertung mit allen Simulationen
	finalRoI := simulateGames(numSimulations, bestConfig.Frequencies, bestConfig.PayoutFactors)
	bestConfig.ExpectedRoI = finalRoI
	
	fmt.Printf("Nach Feintuning: RoI = %.4f%% (%.2f%% Abweichung)\n", 
		finalRoI*100, math.Abs(finalRoI-targetRoI)*100)

	return bestConfig
}

func printConfig(config SlotConfig) {
	fmt.Println("\n" + "=====================")
	fmt.Println("OPTIMALE KONFIGURATION")
	fmt.Println("=======================")
	
	fmt.Printf("Erwartete RoI: %.4f%% (Abweichung vom Ziel: %.2f%%)\n", 
		config.ExpectedRoI*100, math.Abs(config.ExpectedRoI-0.98)*100)
	
	fmt.Println("\nSymbol-Häufigkeiten:")
	fmt.Print("symbolFrequencies = []int{")
	for i, freq := range config.Frequencies {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%d", freq)
	}
	fmt.Println("}")
	
	fmt.Println("\nAuszahlungsfaktoren:")
	fmt.Println("payoutFactors = map[string]float32{")
	
	// Gruppiere die Ausgabe für bessere Lesbarkeit
	groups := map[string][]string{
		"Standard": {},
		"Joker": {},
		"Money Bag": {},
	}
	
	for combo, factor := range config.PayoutFactors {
		// Formatiere Dezimalzahlen schön (entferne .0 bei ganzen Zahlen)
		var factorStr string
		if factor == float32(int(factor)) {
			factorStr = fmt.Sprintf("%.0f", factor)
		} else {
			factorStr = fmt.Sprintf("%.1f", factor)
		}
		
		line := fmt.Sprintf("\t\"%s\": %s,", combo, factorStr)
		
		if combo[len(combo)-1:] == "❓" || combo[0:3] == "❓" {
			groups["Joker"] = append(groups["Joker"], line)
		} else if combo[len(combo)-3:] == "💰" || combo[0:3] == "💰" {
			groups["Money Bag"] = append(groups["Money Bag"], line)
		} else {
			groups["Standard"] = append(groups["Standard"], line)
		}
	}
	
	for groupName, lines := range groups {
		if len(lines) > 0 {
			fmt.Printf("\n\t// %s Kombinationen\n", groupName)
			for _, line := range lines {
				fmt.Println(line)
			}
		}
	}
	
	fmt.Println("}")
}

func main() {
	fmt.Println("🎰 Slot Machine RoI Optimizer")
	fmt.Println("Ziel: Return on Investment von 98%")
	fmt.Println("Optimiert: Symbol-Häufigkeiten + Auszahlungsfaktoren (gerundet auf 1 Nachkommastelle)")
	
	targetRoI := 1.02
	numSimulations := 5000000
	
	fmt.Printf("\nSimuliere mit %d Spielen...\n", numSimulations)
	
	startTime := time.Now()
	optimalConfig := optimizeRoI(targetRoI, numSimulations)
	duration := time.Since(startTime)
	
	printConfig(optimalConfig)
	
	fmt.Printf("\nOptimierung abgeschlossen in %.2f Sekunden\n", duration.Seconds())
	
	// Verifikation mit noch mehr Simulationen
	fmt.Println("\n=== VERIFIKATION ===")
	verificationGames := 5000000
	fmt.Printf("Verifikation mit %d Spielen...\n", verificationGames)
	
	verificationRoI := simulateGames(verificationGames, optimalConfig.Frequencies, optimalConfig.PayoutFactors)
	fmt.Printf("Verifikations-RoI: %.4f%% (%.2f%% Abweichung vom Ziel)\n", 
		verificationRoI*100, math.Abs(verificationRoI-targetRoI)*100)
}