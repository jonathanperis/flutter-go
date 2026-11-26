package characters

// Port of lib/src/grapheme_clusters/constants.dart (package:characters
// 1.4.1): the Unicode grapheme breaking algorithm's character categories
// and the automaton states.

// Unicode Grapheme Breaking Algorithm Character Categories.
const (
	categoryCR                  = 0
	categoryControl             = 1
	categoryOther               = 2 // Any character not in any other category.
	categoryExtend              = 3
	categorySpacingMark         = 4
	categoryRegionalIndicator   = 5
	categoryPictographic        = 6
	categoryLF                  = 7
	categoryPrepend             = 8
	categoryL                   = 9
	categoryV                   = 10
	categoryT                   = 11
	categoryLV                  = 12
	categoryLVT                 = 13
	categoryOtherIndicConsonant = 14 // Other + InCB=Consonant.
	categoryZWJ                 = 15 // Is also InCB=Extend.
	categoryExtendIndicExtend   = 16 // Extend + InCB=Extend.
	categoryExtendIndicLinked   = 17 // Extend + InCB=Linked.
	categoryEoT                 = 18 // End of Text (synthetic input)

	categoryCount      = categoryEoT + 1
	inputCategoryCount = categoryEoT

	regionalIndicatorStart = 0x1F1E6 // A
	regionalIndicatorEnd   = 0x1F1FF // Z
)

// Automaton flags.
const (
	// flagNoBreak/flagBreak are or'ed to the automaton output: whether there
	// should be a break before the most recent input character.
	flagNoBreak = 0
	flagBreak   = 1
	maskBreak   = 1

	// flagLookahead triggers or modifies the effect of lookahead/lookbehind.
	flagLookahead = 2
	maskLookahead = 2

	maskFlags = maskLookahead | maskBreak
	maskState = ^maskFlags

	flagLookaheadBreakNone  = flagNoBreak
	flagLookaheadBreakEarly = flagBreak
	flagLookaheadBreakLate  = flagLookahead | flagNoBreak // Not used.
	flagLookaheadBreakBoth  = flagLookahead | flagBreak

	// automatonRowLength is the number of input categories rounded up to a
	// multiple of maskFlags+1; all state integers are multiples of it.
	automatonRowLength = (categoryCount + maskFlags) & maskState
)

// States of the forwards automaton.
const (
	stateBreak           = 0x00 * automatonRowLength // Always break before next.
	stateCR              = 0x01 * automatonRowLength // Break unless next is LF.
	stateOther           = 0x02 * automatonRowLength // Break unless next is Extend, ZWJ, SpacingMark.
	statePrepend         = 0x03 * automatonRowLength // Break only if next is Control/CR/LF/eot.
	stateL               = 0x04 * automatonRowLength
	stateV               = 0x05 * automatonRowLength
	stateT               = 0x06 * automatonRowLength
	statePictographic    = 0x07 * automatonRowLength
	statePictographicZWJ = 0x08 * automatonRowLength
	stateRegionalSingle  = 0x09 * automatonRowLength
	stateInC             = 0x0A * automatonRowLength
	stateInCL            = 0x0B * automatonRowLength
	stateSoTNoBreak      = 0x0C * automatonRowLength
	stateSoT             = 0x0D * automatonRowLength

	// Context-unaware states.
	stateCAny = 0x0E * automatonRowLength
	stateCZWJ = 0x0F * automatonRowLength
	stateCIE  = 0x10 * automatonRowLength
	stateCIL  = 0x11 * automatonRowLength
	stateCIEZ = 0x12 * automatonRowLength
	stateCILZ = 0x13 * automatonRowLength
	stateCZIE = 0x14 * automatonRowLength
	stateCZIL = 0x15 * automatonRowLength
	stateCExt = 0x16 * automatonRowLength
	stateCExZ = 0x17 * automatonRowLength
	stateCReg = 0x18 * automatonRowLength

	stateMinContextUnaware = stateCAny
	stateLimit             = stateCReg + automatonRowLength
)

// Backwards automaton extra/alternative states and categories.
const (
	categorySoT = categoryEoT // Start of Text (synthetic input)

	stateEoT          = stateSoT
	stateLF           = stateCR
	stateExtend       = statePrepend
	stateEoTNoBreak   = stateSoTNoBreak
	stateRegionalEven = stateInCL
	stateRegionalOdd  = statePictographicZWJ
	stateLookaheadMin = stateLookaheadZWJPictographic

	stateLookaheadZWJPictographic = 0x0E * automatonRowLength
	stateLookaheadInC             = 0x0F * automatonRowLength
	stateLookaheadInCL            = 0x10 * automatonRowLength
	stateLookaheadRegionalEven    = 0x11 * automatonRowLength
	stateLookaheadRegionalOdd     = 0x12 * automatonRowLength

	backStateLimit = stateLookaheadRegionalOdd + automatonRowLength
)
