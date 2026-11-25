package foundation

// Unicode holds the bidirectional formatting characters Flutter names
// (Dart's Unicode is a class of static constants).
var Unicode = struct {
	LRE, RLE, PDF, LRO, RLO, LRI, RLI, FSI, PDI, LRM, RLM, ALM string
}{
	LRE: "‪", RLE: "‫", PDF: "‬", LRO: "‭", RLO: "‮",
	LRI: "⁦", RLI: "⁧", FSI: "⁨", PDI: "⁩",
	LRM: "‎", RLM: "‏", ALM: "؜",
}
