package handler

var (
	symbols = []string{"❌", "❓", "🍒", "🍋", "🍊", "🍇", "⭐", "💎", "💰"}
	symbolFrequencies = []int{10, 13, 25, 20, 15, 13, 9, 5, 1} // Häufigkeiten anpassbar
	payoutFactors = map[string]float32{

		// Standart Kombis
		"❓❓❓": 3,
		"🍒🍒🍒": 4,
		"🍋🍋🍋": 5,
		"🍊🍊🍊": 10,
		"🍇🍇🍇": 20,
		"⭐⭐⭐": 40,
		"💎💎💎": 100,
		"💰💰💰": 500,

		// Erweiterte Kombinationen mit ?
		"🍒🍒❓": 0.8,
		"🍒❓🍒": 0.8,
		"❓🍒🍒": 0.8,
		"🍋🍋❓": 1,
		"🍋❓🍋": 1,
		"❓🍋🍋": 1,
		"🍊🍊❓": 1.5,
		"🍊❓🍊": 1.5,
		"❓🍊🍊": 1.5,
		"🍇🍇❓": 3,
		"🍇❓🍇": 3,
		"❓🍇🍇": 3,
		"⭐⭐❓": 5,
		"⭐❓⭐": 5,
		"❓⭐⭐": 5,
		"💎💎❓": 10,
		"💎❓💎": 10,
		"❓💎💎": 10,
		"💰💰❓": 20,
		"💰❓💰": 20,
		"❓💰💰": 20,

		// Erweiterung mit Money Bag
		"💰💰🍒": 40,
		"💰🍒💰": 40,
		"🍒💰💰": 40,
		"💰💰🍋": 50,
		"💰🍋💰": 50,
		"🍋💰💰": 50,
		"💰💰🍊": 60,
		"💰🍊💰": 60,
		"🍊💰💰": 60,
		"💰💰🍇": 80,
		"💰🍇💰": 80,
		"🍇💰💰": 80,
		"💰💰⭐": 100,
		"💰⭐💰": 100,
		"⭐💰💰": 100,
		"💰💰💎": 200,
		"💰💎💰": 200,
		"💎💰💰": 200,

		"❓❓💰": 10,
		"❓💰❓": 10,
		"💰❓❓": 10,
		"🍒🍒💰": 20,
		"🍒💰🍒": 20,
		"💰🍒🍒": 20,
		"🍋🍋💰": 25,
		"🍋💰🍋": 25,
		"💰🍋🍋": 25,
		"🍊🍊💰": 30,
		"🍊💰🍊": 30,
		"💰🍊🍊": 30,
		"🍇🍇💰": 40,
		"🍇💰🍇": 40,
		"💰🍇🍇": 40,
		"⭐⭐💰": 50,
		"⭐💰⭐": 50,
		"💰⭐⭐": 50,
		"💎💎💰": 100,
		"💎💰💎": 100,
		"💰💎💎": 100,
	})