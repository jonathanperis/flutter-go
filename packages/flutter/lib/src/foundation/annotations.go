package foundation

// Port of annotations.dart (Flutter 3.50.1): documentation annotations.
// Go has no annotations; the classes are kept as plain values (a doc tool
// could read them from a package-level variable), with Dart's fields.

// Category is a category with which to annotate a class, for documentation
// purposes: the sections, from the most general to the most specific.
type Category struct {
	Sections []string
}

// DocumentationIcon is an image with which to annotate a class, for
// documentation purposes: its URL.
type DocumentationIcon struct {
	URL string
}

// Summary is a short summary of a class, for documentation purposes.
type Summary struct {
	Text string
}
