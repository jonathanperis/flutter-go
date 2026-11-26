package characters

// Port of lib/src/extensions.dart (package:characters 1.4.1).

// StringCharacters is the extension StringCharacters on String: its one
// member, the getter `characters`, is this function (Dart's
// `string.characters`).
func StringCharacters(s string) Characters { return NewCharacters(s) }
