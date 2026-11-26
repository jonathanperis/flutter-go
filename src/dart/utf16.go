package dart

// UTF16CodeUnits is String.codeUnits: the UTF-16 code units of a Go string
// read as WTF-8 (see utf8.go), so a lone surrogate is kept as Dart's string
// would hold it.
func UTF16CodeUnits(s string) []uint16 { return utf16Units(s) }
