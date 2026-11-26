// The text differential test (../text_test.dart) run again under
// --enable-impeller, which writes build/text_impeller.json: paragraph layout
// is SkParagraph's on both backends, and Go checks both references.
import '../text_test.dart' as text;

void main() => text.main();
