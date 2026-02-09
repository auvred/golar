package vue_parser

import "unicode/utf8"

type ErrorCode = uint32

const (
	// parse errors
	ErrorCode_ABRUPT_CLOSING_OF_EMPTY_COMMENT ErrorCode = iota
	ErrorCode_CDATA_IN_HTML_CONTENT
	ErrorCode_DUPLICATE_ATTRIBUTE
	ErrorCode_END_TAG_WITH_ATTRIBUTES
	ErrorCode_END_TAG_WITH_TRAILING_SOLIDUS
	ErrorCode_EOF_BEFORE_TAG_NAME
	ErrorCode_EOF_IN_CDATA
	ErrorCode_EOF_IN_COMMENT
	ErrorCode_EOF_IN_SCRIPT_HTML_COMMENT_LIKE_TEXT
	ErrorCode_EOF_IN_TAG
	ErrorCode_INCORRECTLY_CLOSED_COMMENT
	ErrorCode_INCORRECTLY_OPENED_COMMENT
	ErrorCode_INVALID_FIRST_CHARACTER_OF_TAG_NAME
	ErrorCode_MISSING_ATTRIBUTE_VALUE
	ErrorCode_MISSING_END_TAG_NAME
	ErrorCode_MISSING_WHITESPACE_BETWEEN_ATTRIBUTES
	ErrorCode_NESTED_COMMENT
	ErrorCode_UNEXPECTED_CHARACTER_IN_ATTRIBUTE_NAME
	ErrorCode_UNEXPECTED_CHARACTER_IN_UNQUOTED_ATTRIBUTE_VALUE
	ErrorCode_UNEXPECTED_EQUALS_SIGN_BEFORE_ATTRIBUTE_NAME
	ErrorCode_UNEXPECTED_NULL_CHARACTER
	ErrorCode_UNEXPECTED_QUESTION_MARK_INSTEAD_OF_TAG_NAME
	ErrorCode_UNEXPECTED_SOLIDUS_IN_TAG

	// Vue-specific parse errors
	ErrorCode_X_INVALID_END_TAG
	ErrorCode_X_MISSING_END_TAG
	ErrorCode_X_MISSING_INTERPOLATION_END
	ErrorCode_X_MISSING_DIRECTIVE_NAME
	ErrorCode_X_MISSING_DYNAMIC_DIRECTIVE_ARGUMENT_END

	// transform errors
	ErrorCode_X_V_IF_NO_EXPRESSION
	ErrorCode_X_V_IF_SAME_KEY
	ErrorCode_X_V_ELSE_NO_ADJACENT_IF
	ErrorCode_X_V_FOR_NO_EXPRESSION
	ErrorCode_X_V_FOR_MALFORMED_EXPRESSION
	ErrorCode_X_V_FOR_TEMPLATE_KEY_PLACEMENT
	ErrorCode_X_V_BIND_NO_EXPRESSION
	ErrorCode_X_V_ON_NO_EXPRESSION
	ErrorCode_X_V_SLOT_UNEXPECTED_DIRECTIVE_ON_SLOT_OUTLET
	ErrorCode_X_V_SLOT_MIXED_SLOT_USAGE
	ErrorCode_X_V_SLOT_DUPLICATE_SLOT_NAMES
	ErrorCode_X_V_SLOT_EXTRANEOUS_DEFAULT_SLOT_CHILDREN
	ErrorCode_X_V_SLOT_MISPLACED
	ErrorCode_X_V_MODEL_NO_EXPRESSION
	ErrorCode_X_V_MODEL_MALFORMED_EXPRESSION
	ErrorCode_X_V_MODEL_ON_SCOPE_VARIABLE
	ErrorCode_X_V_MODEL_ON_PROPS
	ErrorCode_X_INVALID_EXPRESSION
	ErrorCode_X_KEEP_ALIVE_INVALID_CHILDREN

	// generic errors
	ErrorCode_X_PREFIX_ID_NOT_SUPPORTED
	ErrorCode_X_MODULE_MODE_NOT_SUPPORTED
	ErrorCode_X_CACHE_HANDLER_NOT_SUPPORTED
	ErrorCode_X_SCOPE_ID_NOT_SUPPORTED
	ErrorCode_X_VNODE_HOOKS

	// placed here to preserve order for the current minor
	// TODO adjust order in 3.5
	ErrorCode_X_V_BIND_INVALID_SAME_NAME_ARGUMENT

	// Special value for higher-order compilers to pick up the last code
	// to avoid collision of error codes. This should always be kept as the last
	// item.
	ErrorCode___EXTEND_POINT__
)

type ParseMode uint8

const (
	ParseModeBase ParseMode = iota
	ParseModeHtml
	ParseModeSfc
)

const (
	CharCodeTab             = 0x9  // "\t"
	CharCodeNewLine         = 0xa  // "\n"
	CharCodeFormFeed        = 0xc  // "\f"
	CharCodeCarriageReturn  = 0xd  // "\r"
	CharCodeSpace           = 0x20 // " "
	CharCodeExclamationMark = 0x21 // "!"
	CharCodeNumber          = 0x23 // "#"
	CharCodeAmp             = 0x26 // "&"
	CharCodeSingleQuote     = 0x27 // "'"
	CharCodeDoubleQuote     = 0x22 // '"'
	CharCodeGraveAccent     = 96   // "`"
	CharCodeDash            = 0x2d // "-"
	CharCodeSlash           = 0x2f // "/"
	CharCodeZero            = 0x30 // "0"
	CharCodeNine            = 0x39 // "9"
	CharCodeSemi            = 0x3b // ";"
	CharCodeLt              = 0x3c // "<"
	CharCodeEq              = 0x3d // "="
	CharCodeGt              = 0x3e // ">"
	CharCodeQuestionmark    = 0x3f // "?"
	CharCodeUpperA          = 0x41 // "A"
	CharCodeLowerA          = 0x61 // "a"
	CharCodeUpperF          = 0x46 // "F"
	CharCodeLowerF          = 0x66 // "f"
	CharCodeUpperZ          = 0x5a // "Z"
	CharCodeLowerZ          = 0x7a // "z"
	CharCodeLowerX          = 0x78 // "x"
	CharCodeLowerV          = 0x76 // "v"
	CharCodeDot             = 0x2e // "."
	CharCodeColon           = 0x3a // ":"
	CharCodeAt              = 0x40 // "@"
	CharCodeLeftSquare      = 91   // "["
	CharCodeRightSquare     = 93   // "]"
)

type State = uint32

const (
	StateText State = iota
	// interpolation
	StateInterpolationOpen
	StateInterpolation
	StateInterpolationClose

	// Tags
	StateBeforeTagName // After <
	StateInTagName
	StateInSelfClosingTag
	StateBeforeClosingTagName
	StateInClosingTagName
	StateAfterClosingTagName

	// Attrs
	StateBeforeAttrName
	StateInAttrName
	StateInDirName
	StateInDirArg
	StateInDirDynamicArg
	StateInDirModifier
	StateAfterAttrName
	StateBeforeAttrValue
	StateInAttrValueDq // "
	StateInAttrValueSq // '
	StateInAttrValueNq

	// Declarations
	StateBeforeDeclaration // !
	StateInDeclaration

	// Processing instructions
	StateInProcessingInstruction // ?

	// Comments & CDATA
	StateBeforeComment
	StateCDATASequence
	StateInSpecialComment
	StateInCommentLike

	// Special tags
	StateBeforeSpecialS // Decide if we deal with `<script` or `<style`
	StateBeforeSpecialT // Decide if we deal with `<title` or `<textarea`
	StateSpecialStartSequence
	StateInRCDATA

	StateInEntity

	StateInSFCRootTagName
)

/**
 * HTML only allows ASCII alpha characters (a-z and A-Z) at the beginning of a
 * tag name.
 */
func isTagStartChar(c rune) bool {
	return (c >= CharCodeLowerA && c <= CharCodeLowerZ) ||
		(c >= CharCodeUpperA && c <= CharCodeUpperZ)
}

func isWhitespace(c rune) bool {
	return c == CharCodeSpace ||
		c == CharCodeNewLine ||
		c == CharCodeTab ||
		c == CharCodeFormFeed ||
		c == CharCodeCarriageReturn
}

func isEndOfTagSection(c rune) bool {
	return c == CharCodeSlash || c == CharCodeGt || isWhitespace(c)
}

func toCharCodes(str string) []rune {
	return []rune(str)
}

type QuoteType uint8

const (
	QuoteTypeNoValue QuoteType = iota
	QuoteTypeUnquoted
	QuoteTypeSingle
	QuoteTypeDouble
)

/**
 * Sequences used to match longer strings.
 *
 * We don't have `Script`, `Style`, or `Title` here. Instead, we re-use the *End
 * sequences with an increased offset.
 */
var (
	SequenceCdata       = []rune{'C', 'D', 'A', 'T', 'A', '['}
	SequenceCdataEnd    = []rune{']', ']', '>'}
	SequenceCommentEnd  = []rune{'-', '-', '>'}
	SequenceScriptEnd   = []rune{'<', '/', 's', 'c', 'r', 'i', 'p', 't'}
	SequenceStyleEnd    = []rune{'<', '/', 's', 't', 'y', 'l', 'e'}
	SequenceTitleEnd    = []rune{'<', '/', 't', 'i', 't', 'l', 'e'}
	SequenceTextareaEnd = []rune{'<', '/', 't', 'e', 'x', 't', 'a', 'r', 'e', 'a'}
)

var (
	defaultDelimitersOpen  = []rune{'{', '{'}
	defaultDelimitersClose = []rune{'}', '}'}
)

type Tokenizer struct {
	/** The current state the tokenizer is in. */
	state State
	/** The read buffer. */
	buffer string
	/** The beginning of the section that is currently being read. */
	sectionStart int
	/** The index within the buffer that we are currently looking at. */
	index int
	/** The start of the last entity. */
	entityStart int
	/** Some behavior, eg. when decoding entities, is done while we are in another state. This keeps track of the other state type. */
	baseState State
	/** For special parsing behavior inside of script and style tags. */
	inRCDATA bool
	/** For disabling RCDATA tags handling */
	inXML bool
	/** For disabling interpolation parsing in v-pre */
	inVPre bool
	/** Record newline positions for fast line / column calculation */
	newlines []int

	// TODO
	// private readonly entityDecoder?: EntityDecoder

	mode ParseMode

	currentSequence *[]rune
	sequenceIndex   int

	// TODO(perf): avoid indirection?
	parser *Parser

	delimiterIndex int

	delimiterOpen  []rune
	delimiterClose []rune

	// constructor(
	//   private readonly stack: ElementNode[],
	//   private readonly cbs: Callbacks,
	// ) {
	//   if (!__BROWSER__) {
	//     t.entityDecoder = new EntityDecoder(htmlDecodeTree, (cp, consumed) =>
	//       t.emitCodePoint(cp, consumed),
	//     )
	//   }
	// }
}

func (t *Tokenizer) inSFCRoot() bool {
	return t.mode == ParseModeSfc && len(t.parser.stack) == 0
}

func NewTokenizer(source string) *Tokenizer {
	return &Tokenizer{
		state:           StateText,
		mode:            ParseModeSfc,
		buffer:          source,
		sectionStart:    0,
		index:           0,
		baseState:       StateText,
		inRCDATA:        false,
		currentSequence: nil,
		newlines:        []int{},
		delimiterIndex:  -1,
		delimiterOpen:   defaultDelimitersOpen,
		delimiterClose:  defaultDelimitersClose,
	}
}

/**
 * Generate Position object with line / column information using recorded
 * newline positions. We know the index is always going to be an already
 * processed index, so all the newlines up to this index should have been
 * recorded.
 */
// func (t *Tokenizer) getPos(index int): Position {
//   let line = 1
//   let column = index + 1
//   for (let i = t.newlines.length - 1; i >= 0; i--) {
//     const newlineIndex = t.newlines[i]
//     if (index > newlineIndex) {
//       line = i + 2
//       column = index - newlineIndex
//       break
//     }
//   }
//   return {
//     column,
//     line,
//     offset: index,
//   }
// }

func (t *Tokenizer) peek() byte {
	if t.index+1 >= len(t.buffer) {
		return 0
	}
	return t.buffer[t.index+1]
}

func (t *Tokenizer) stateText(c rune) {
	if c == CharCodeLt {
		if t.index > t.sectionStart {
			t.parser.ontext(t.sectionStart, t.index)
		}
		t.state = StateBeforeTagName
		t.sectionStart = t.index
	// } else if c == CharCodeAmp {
	// 	t.startEntity()
	} else if !t.inVPre && c == defaultDelimitersOpen[0] {
		t.state = StateInterpolationOpen
		t.delimiterIndex = 0
		t.stateInterpolationOpen(c)
	}
}

// TODO: delimiters can only be ASCII, use unicode/utf8 instead
func (t *Tokenizer) stateInterpolationOpen(c rune) {
	if c == t.delimiterOpen[t.delimiterIndex] {
		if t.delimiterIndex == len(t.delimiterOpen)-1 {
			start := t.index + 1 - len(t.delimiterOpen)
			if start > t.sectionStart {
				t.parser.ontext(t.sectionStart, start)
			}
			t.state = StateInterpolation
			t.sectionStart = start
		} else {
			t.delimiterIndex++
		}
	} else if t.inRCDATA {
		t.state = StateInRCDATA
		t.stateInRCDATA(c)
	} else {
		t.state = StateText
		t.stateText(c)
	}
}

func (t *Tokenizer) stateInterpolation(c rune) {
	if c == t.delimiterClose[0] {
		t.state = StateInterpolationClose
		t.delimiterIndex = 0
		t.stateInterpolationClose(c)
	}
}

func (t *Tokenizer) stateInterpolationClose(c rune) {
	if c == t.delimiterClose[t.delimiterIndex] {
		if t.delimiterIndex == len(t.delimiterClose)-1 {
			t.parser.oninterpolation(t.sectionStart, t.index+1)
			if t.inRCDATA {
				t.state = StateInRCDATA
			} else {
				t.state = StateText
			}
			t.sectionStart = t.index + 1
		} else {
			t.delimiterIndex++
		}
	} else {
		t.state = StateInterpolation
		t.stateInterpolation(c)
	}
}

// public currentSequence []byte = undefined!
// private sequenceIndex = 0
func (t *Tokenizer) stateSpecialStartSequence(c rune) {
	isEnd := t.sequenceIndex == len(*t.currentSequence)
	var isMatch bool
	if isEnd {
		// If we are at the end of the sequence, make sure the tag name has ended
		isMatch = isEndOfTagSection(c)
	} else {
		// Otherwise, do a case-insensitive comparison
		isMatch = (c | 0x20) == (*t.currentSequence)[t.sequenceIndex]
	}

	if !isMatch {
		t.inRCDATA = false
	} else if !isEnd {
		t.sequenceIndex++
		return
	}

	t.sequenceIndex = 0
	t.state = StateInTagName
	t.stateInTagName(c)
}

/** Look for an end tag. For <title> and <textarea>, also decode entities. */
func (t *Tokenizer) stateInRCDATA(c rune) {
	if t.sequenceIndex == len(*t.currentSequence) {
		if c == CharCodeGt || isWhitespace(c) {
			endOfText := t.index - len(*t.currentSequence)

			if t.sectionStart < endOfText {
				// Spoof the index so that reported locations match up.
				actualIndex := t.index
				t.index = endOfText
				t.parser.ontext(t.sectionStart, endOfText)
				t.index = actualIndex
			}

			t.sectionStart = endOfText + 2 // Skip over the `</`
			t.stateInClosingTagName(c)
			t.inRCDATA = false
			return // We are done; skip the rest of the function.
		}

		t.sequenceIndex = 0
	}

	if (c | 0x20) == (*t.currentSequence)[t.sequenceIndex] {
		t.sequenceIndex += 1
	} else if t.sequenceIndex == 0 {
		if t.currentSequence == &SequenceTitleEnd || (t.currentSequence == &SequenceTextareaEnd && !t.inSFCRoot()) {
			// We have to parse entities in <title> and <textarea> tags.
			// if c == CharCodeAmp {
			// 	t.startEntity()
			// } else
			if !t.inVPre && c == t.delimiterOpen[0] {
				// We also need to handle interpolation
				t.state = StateInterpolationOpen
				t.delimiterIndex = 0
				t.stateInterpolationOpen(c)
			}
		} else if t.fastForwardTo(CharCodeLt) {
			// Outside of <title> and <textarea> tags, we can fast-forward.
			t.sequenceIndex = 1
		}
	} else {
		// If we see a `<`, set the sequence index to 1; useful for eg. `<</script>`.
		if c == CharCodeLt {
			t.sequenceIndex = 1
		} else {
			t.sequenceIndex = 0
		}
	}
}

func (t *Tokenizer) stateCDATASequence(c rune) {
	if c == SequenceCdata[t.sequenceIndex] {
		t.sequenceIndex++
		if t.sequenceIndex == len(SequenceCdata) {
			t.state = StateInCommentLike
			t.currentSequence = &SequenceCdataEnd
			t.sequenceIndex = 0
			t.sectionStart = t.index + 1
		}
	} else {
		t.sequenceIndex = 0
		t.state = StateInDeclaration
		t.stateInDeclaration(c) // Reconsume the character
	}
}

/**
 * When we wait for one specific character, we can speed things up
 * by skipping through the buffer until we find it.
 *
 * @returns Whether the character was found.
 */
// TODO: non-ASCII
func (t *Tokenizer) fastForwardTo(c rune) bool {
	t.index++
	for ; t.index < len(t.buffer); t.index++ {
		cc := t.buffer[t.index]
		if cc == CharCodeNewLine {
			// TODO:
			// t.newlines.push(t.index)
		}
		if rune(cc) == c {
			return true
		}
	}

	/*
	 * We increment the index at the end of the `parse` loop,
	 * so set it to `buffer.length - 1` here.
	 *
	 * TODO: Refactor `parse` to increment index before calling states.
	 */
	t.index = len(t.buffer) - 1

	return false
}

/**
 * Comments and CDATA end with `-->` and `]]>`.
 *
 * Their common qualities are:
 * - Their end sequences have a distinct character they start with.
 * - That character is then repeated, so we have to check multiple repeats.
 * - All characters but the start character of the sequence can be skipped.
 */
func (t *Tokenizer) stateInCommentLike(c rune) {
	if c == (*t.currentSequence)[t.sequenceIndex] {
		t.sequenceIndex++
		if t.sequenceIndex == len(*t.currentSequence) {
			if t.currentSequence == &SequenceCdataEnd {
				t.parser.oncdata(t.sectionStart, t.index-2)
			} else {
				t.parser.oncomment(t.sectionStart, t.index-2)
			}

			t.sequenceIndex = 0
			t.sectionStart = t.index + 1
			t.state = StateText
		}
	} else if t.sequenceIndex == 0 {
		// Fast-forward to the first character of the sequence
		if t.fastForwardTo((*t.currentSequence)[0]) {
			t.sequenceIndex = 1
		}
	} else if c != (*t.currentSequence)[t.sequenceIndex-1] {
		// Allow long sequences, eg. --->, ]]]>
		t.sequenceIndex = 0
	}
}

func (t *Tokenizer) startSpecial(sequence *[]rune, offset int) {
	t.enterRCDATA(sequence, offset)
	t.state = StateSpecialStartSequence
}

func (t *Tokenizer) enterRCDATA(sequence *[]rune, offset int) {
	t.inRCDATA = true
	t.currentSequence = sequence
	t.sequenceIndex = offset
}

func (t *Tokenizer) stateBeforeTagName(c rune) {
	if c == CharCodeExclamationMark {
		t.state = StateBeforeDeclaration
		t.sectionStart = t.index + 1
	} else if c == CharCodeQuestionmark {
		t.state = StateInProcessingInstruction
		t.sectionStart = t.index + 1
	} else if isTagStartChar(c) {
		t.sectionStart = t.index
		if t.mode == ParseModeBase {
			// no special tags in base mode
			t.state = StateInTagName
		} else if t.inSFCRoot() {
			// SFC mode + root level
			// - everything except <template> is RAWTEXT
			// - <template> with lang other than html is also RAWTEXT
			t.state = StateInSFCRootTagName
		} else if !t.inXML {
			// HTML mode
			// - <script>, <style> RAWTEXT
			// - <title>, <textarea> RCDATA
			if c == 116 /* t */ {
				t.state = StateBeforeSpecialT
			} else {
				if c == 115 /* s */ {
					t.state = StateBeforeSpecialS
				} else {
					t.state = StateInTagName
				}
			}
		} else {
			t.state = StateInTagName
		}
	} else if c == CharCodeSlash {
		t.state = StateBeforeClosingTagName
	} else {
		t.state = StateText
		t.stateText(c)
	}
}
func (t *Tokenizer) stateInTagName(c rune) {
	if isEndOfTagSection(c) {
		t.handleTagName(c)
	}
}
func (t *Tokenizer) stateInSFCRootTagName(c rune) {
	if isEndOfTagSection(c) {
		tag := t.buffer[t.sectionStart:t.index]
		if tag != "template" {
			d := toCharCodes(`</` + tag)
			t.enterRCDATA(&d, 0)
		}
		t.handleTagName(c)
	}
}
func (t *Tokenizer) handleTagName(c rune) {
	t.parser.onopentagname(t.sectionStart, t.index)
	t.sectionStart = -1
	t.state = StateBeforeAttrName
	t.stateBeforeAttrName(c)
}
func (t *Tokenizer) stateBeforeClosingTagName(c rune) {
	if isWhitespace(c) {
		// Ignore
	} else if c == CharCodeGt {
		t.parser.onerr(ErrorCode_MISSING_END_TAG_NAME, t.index)
		t.state = StateText
		// Ignore
		t.sectionStart = t.index + 1
	} else {
		if isTagStartChar(c) {
			t.state = StateInClosingTagName
		} else {
			t.state = StateInSpecialComment
		}
		t.sectionStart = t.index
	}
}
func (t *Tokenizer) stateInClosingTagName(c rune) {
	if c == CharCodeGt || isWhitespace(c) {
		t.parser.onclosetag(t.sectionStart, t.index)
		t.sectionStart = -1
		t.state = StateAfterClosingTagName
		t.stateAfterClosingTagName(c)
	}
}
func (t *Tokenizer) stateAfterClosingTagName(c rune) {
	// Skip everything until ">"
	if c == CharCodeGt {
		t.state = StateText
		t.sectionStart = t.index + 1
	}
}
func (t *Tokenizer) stateBeforeAttrName(c rune) {
	if c == CharCodeGt {
		t.parser.onopentagend(t.index)
		if t.inRCDATA {
			t.state = StateInRCDATA
		} else {
			t.state = StateText
		}
		t.sectionStart = t.index + 1
	} else if c == CharCodeSlash {
		t.state = StateInSelfClosingTag
		// TODO:
		if /* __DEV__ || */ t.peek() != CharCodeGt {
			t.parser.onerr(ErrorCode_UNEXPECTED_SOLIDUS_IN_TAG, t.index)
		}
	} else if c == CharCodeLt && t.peek() == CharCodeSlash {
		// special handling for </ appearing in open tag state
		// this is different from standard HTML parsing but makes practical sense
		// especially for parsing intermediate input state in IDEs.
		t.parser.onopentagend(t.index)
		t.state = StateBeforeTagName
		t.sectionStart = t.index
	} else if !isWhitespace(c) {
		if c == CharCodeEq {
			t.parser.onerr(
				ErrorCode_UNEXPECTED_EQUALS_SIGN_BEFORE_ATTRIBUTE_NAME,
				t.index,
			)
		}
		t.handleAttrStart(c)
	}
}
func (t *Tokenizer) handleAttrStart(c rune) {
	if c == CharCodeLowerV && t.peek() == CharCodeDash {
		t.state = StateInDirName
		t.sectionStart = t.index
	} else if c == CharCodeDot || c == CharCodeColon || c == CharCodeAt || c == CharCodeNumber {
		t.parser.ondirname(t.index, t.index+1)
		t.state = StateInDirArg
		t.sectionStart = t.index + 1
	} else {
		t.state = StateInAttrName
		t.sectionStart = t.index
	}
}
func (t *Tokenizer) stateInSelfClosingTag(c rune) {
	if c == CharCodeGt {
		t.parser.onselfclosingtag(t.index)
		t.state = StateText
		t.sectionStart = t.index + 1
		t.inRCDATA = false // Reset special state, in case of self-closing special tags
	} else if !isWhitespace(c) {
		t.state = StateBeforeAttrName
		t.stateBeforeAttrName(c)
	}
}
func (t *Tokenizer) stateInAttrName(c rune) {
	if c == CharCodeEq || isEndOfTagSection(c) {
		t.parser.onattribname(t.sectionStart, t.index)
		t.handleAttrNameEnd(c)
	} else if c == CharCodeDoubleQuote || c == CharCodeSingleQuote || c == CharCodeLt {
		t.parser.onerr(
			ErrorCode_UNEXPECTED_CHARACTER_IN_ATTRIBUTE_NAME,
			t.index,
		)
	}
}
func (t *Tokenizer) stateInDirName(c rune) {
	if c == CharCodeEq || isEndOfTagSection(c) {
		t.parser.ondirname(t.sectionStart, t.index)
		t.handleAttrNameEnd(c)
	} else if c == CharCodeColon {
		t.parser.ondirname(t.sectionStart, t.index)
		t.state = StateInDirArg
		t.sectionStart = t.index + 1
	} else if c == CharCodeDot {
		t.parser.ondirname(t.sectionStart, t.index)
		t.state = StateInDirModifier
		t.sectionStart = t.index + 1
	}
}
func (t *Tokenizer) stateInDirArg(c rune) {
	if c == CharCodeEq || isEndOfTagSection(c) {
		t.parser.ondirarg(t.sectionStart, t.index)
		t.handleAttrNameEnd(c)
	} else if c == CharCodeLeftSquare {
		t.state = StateInDirDynamicArg
	} else if c == CharCodeDot {
		t.parser.ondirarg(t.sectionStart, t.index)
		t.state = StateInDirModifier
		t.sectionStart = t.index + 1
	}
}
func (t *Tokenizer) stateInDynamicDirArg(c rune) {
	if c == CharCodeRightSquare {
		t.state = StateInDirArg
	} else if c == CharCodeEq || isEndOfTagSection(c) {
		t.parser.ondirarg(t.sectionStart, t.index+1)
		t.handleAttrNameEnd(c)
		t.parser.onerr(
			ErrorCode_X_MISSING_DYNAMIC_DIRECTIVE_ARGUMENT_END,
			t.index,
		)

	}
}
func (t *Tokenizer) stateInDirModifier(c rune) {
	if c == CharCodeEq || isEndOfTagSection(c) {
		t.parser.ondirmodifier(t.sectionStart, t.index)
		t.handleAttrNameEnd(c)
	} else if c == CharCodeDot {
		t.parser.ondirmodifier(t.sectionStart, t.index)
		t.sectionStart = t.index + 1
	}
}
func (t *Tokenizer) handleAttrNameEnd(c rune) {
	t.sectionStart = t.index
	t.state = StateAfterAttrName
	t.parser.onattribnameend(t.index)
	t.stateAfterAttrName(c)
}
func (t *Tokenizer) stateAfterAttrName(c rune) {
	if c == CharCodeEq {
		t.state = StateBeforeAttrValue
	} else if c == CharCodeSlash || c == CharCodeGt {
		t.parser.onattribend(QuoteTypeNoValue, t.sectionStart)
		t.sectionStart = -1
		t.state = StateBeforeAttrName
		t.stateBeforeAttrName(c)
	} else if !isWhitespace(c) {
		t.parser.onattribend(QuoteTypeNoValue, t.sectionStart)
		t.handleAttrStart(c)
	}
}
func (t *Tokenizer) stateBeforeAttrValue(c rune) {
	if c == CharCodeDoubleQuote {
		t.state = StateInAttrValueDq
		t.sectionStart = t.index + 1
	} else if c == CharCodeSingleQuote {
		t.state = StateInAttrValueSq
		t.sectionStart = t.index + 1
	} else if !isWhitespace(c) {
		t.sectionStart = t.index
		t.state = StateInAttrValueNq
		t.stateInAttrValueNoQuotes(c) // Reconsume token
	}
}
func (t *Tokenizer) handleInAttrValue(c rune, quote rune) {
	if c == quote {
		t.parser.onattribdata(t.sectionStart, t.index)
		t.sectionStart = -1
		q := QuoteTypeSingle
		if quote == CharCodeDoubleQuote {
			q = QuoteTypeDouble
		}
		t.parser.onattribend(
			q,
			t.index+1,
		)
		t.state = StateBeforeAttrName
	} else if c == CharCodeAmp {
		// t.startEntity()
	}
}
func (t *Tokenizer) stateInAttrValueDoubleQuotes(c rune) {
	t.handleInAttrValue(c, CharCodeDoubleQuote)
}
func (t *Tokenizer) stateInAttrValueSingleQuotes(c rune) {
	t.handleInAttrValue(c, CharCodeSingleQuote)
}
func (t *Tokenizer) stateInAttrValueNoQuotes(c rune) {
	if isWhitespace(c) || c == CharCodeGt {
		t.parser.onattribdata(t.sectionStart, t.index)
		t.sectionStart = -1
		t.parser.onattribend(QuoteTypeUnquoted, t.index)
		t.state = StateBeforeAttrName
		t.stateBeforeAttrName(c)
	} else if (c == CharCodeDoubleQuote) || c == CharCodeSingleQuote || c == CharCodeLt || c == CharCodeEq || c == CharCodeGraveAccent {
		t.parser.onerr(
			ErrorCode_UNEXPECTED_CHARACTER_IN_UNQUOTED_ATTRIBUTE_VALUE,
			t.index,
		)
	} else if c == CharCodeAmp {
		// t.startEntity()
	}
}
func (t *Tokenizer) stateBeforeDeclaration(c rune) {
	if c == CharCodeLeftSquare {
		t.state = StateCDATASequence
		t.sequenceIndex = 0
	} else {
		if c == CharCodeDash {
			t.state = StateBeforeComment
		} else {
			t.state = StateInDeclaration
		}
	}
}
func (t *Tokenizer) stateInDeclaration(c rune) {
	if c == CharCodeGt || t.fastForwardTo(CharCodeGt) {
		// t.parser.ondeclaration(t.sectionStart, t.index)
		t.state = StateText
		t.sectionStart = t.index + 1
	}
}
func (t *Tokenizer) stateInProcessingInstruction(c rune) {
	if c == CharCodeGt || t.fastForwardTo(CharCodeGt) {
		t.parser.onprocessinginstruction(t.sectionStart, t.index)
		t.state = StateText
		t.sectionStart = t.index + 1
	}
}
func (t *Tokenizer) stateBeforeComment(c rune) {
	if c == CharCodeDash {
		t.state = StateInCommentLike
		t.currentSequence = &SequenceCommentEnd
		// Allow short comments (eg. <!-->)
		t.sequenceIndex = 2
		t.sectionStart = t.index + 1
	} else {
		t.state = StateInDeclaration
	}
}
func (t *Tokenizer) stateInSpecialComment(c rune) {
	if c == CharCodeGt || t.fastForwardTo(CharCodeGt) {
		t.parser.oncomment(t.sectionStart, t.index)
		t.state = StateText
		t.sectionStart = t.index + 1
	}
}
func (t *Tokenizer) stateBeforeSpecialS(c rune) {
	if c == SequenceScriptEnd[3] {
		t.startSpecial(&SequenceScriptEnd, 4)
	} else if c == SequenceStyleEnd[3] {
		t.startSpecial(&SequenceStyleEnd, 4)
	} else {
		t.state = StateInTagName
		t.stateInTagName(c) // Consume the token again
	}
}
func (t *Tokenizer) stateBeforeSpecialT(c rune) {
	if c == SequenceTitleEnd[3] {
		t.startSpecial(&SequenceTitleEnd, 4)
	} else if c == SequenceTextareaEnd[3] {
		t.startSpecial(&SequenceTextareaEnd, 4)
	} else {
		t.state = StateInTagName
		t.stateInTagName(c) // Consume the token again
	}
}

func (t *Tokenizer) startEntity() {
	t.baseState = t.state
	t.state = StateInEntity
	t.entityStart = t.index
	// TODO
	// t.entityDecoder.startEntity(
	//   t.baseState == StateText || t.baseState == StateInRCDATA
	//     ? DecodingMode.Legacy
	//     : DecodingMode.Attribute,
	// )

}

func (t *Tokenizer) stateInEntity() {
	// TODO
	// length := t.entityDecoder!.write(t.buffer, t.index)

	// // If `length` is positive, we are done with the entity.
	// if length >= 0 {
	//   t.state = t.baseState
	//
	//   if length == 0 {
	//     t.index = t.entityStart
	//   }
	// } else {
	//   // Mark buffer as consumed.
	//   t.index = len(t.buffer) - 1
	// }

}

/**
 * Iterates through the buffer, calling the function corresponding to the current state.
 *
 * States that are more likely to be hit are higher up, as a performance improvement.
 */
func (t *Tokenizer) parse() {
	for t.index < len(t.buffer) {
		c, size := utf8.DecodeRuneInString(t.buffer[t.index:])
		if c == CharCodeNewLine && t.state != StateInEntity {
			// TODO:
			// t.newlines.push(t.index)
		}
		switch t.state {
		case StateText:
			{
				t.stateText(c)
				break
			}
		case StateInterpolationOpen:
			{
				t.stateInterpolationOpen(c)
				break
			}
		case StateInterpolation:
			{
				t.stateInterpolation(c)
				break
			}
		case StateInterpolationClose:
			{
				t.stateInterpolationClose(c)
				break
			}
		case StateSpecialStartSequence:
			{
				t.stateSpecialStartSequence(c)
				break
			}
		case StateInRCDATA:
			{
				t.stateInRCDATA(c)
				break
			}
		case StateCDATASequence:
			{
				t.stateCDATASequence(c)
				break
			}
		case StateInAttrValueDq:
			{
				t.stateInAttrValueDoubleQuotes(c)
				break
			}
		case StateInAttrName:
			{
				t.stateInAttrName(c)
				break
			}
		case StateInDirName:
			{
				t.stateInDirName(c)
				break
			}
		case StateInDirArg:
			{
				t.stateInDirArg(c)
				break
			}
		case StateInDirDynamicArg:
			{
				t.stateInDynamicDirArg(c)
				break
			}
		case StateInDirModifier:
			{
				t.stateInDirModifier(c)
				break
			}
		case StateInCommentLike:
			{
				t.stateInCommentLike(c)
				break
			}
		case StateInSpecialComment:
			{
				t.stateInSpecialComment(c)
				break
			}
		case StateBeforeAttrName:
			{
				t.stateBeforeAttrName(c)
				break
			}
		case StateInTagName:
			{
				t.stateInTagName(c)
				break
			}
		case StateInSFCRootTagName:
			{
				t.stateInSFCRootTagName(c)
				break
			}
		case StateInClosingTagName:
			{
				t.stateInClosingTagName(c)
				break
			}
		case StateBeforeTagName:
			{
				t.stateBeforeTagName(c)
				break
			}
		case StateAfterAttrName:
			{
				t.stateAfterAttrName(c)
				break
			}
		case StateInAttrValueSq:
			{
				t.stateInAttrValueSingleQuotes(c)
				break
			}
		case StateBeforeAttrValue:
			{
				t.stateBeforeAttrValue(c)
				break
			}
		case StateBeforeClosingTagName:
			{
				t.stateBeforeClosingTagName(c)
				break
			}
		case StateAfterClosingTagName:
			{
				t.stateAfterClosingTagName(c)
				break
			}
		case StateBeforeSpecialS:
			{
				t.stateBeforeSpecialS(c)
				break
			}
		case StateBeforeSpecialT:
			{
				t.stateBeforeSpecialT(c)
				break
			}
		case StateInAttrValueNq:
			{
				t.stateInAttrValueNoQuotes(c)
				break
			}
		case StateInSelfClosingTag:
			{
				t.stateInSelfClosingTag(c)
				break
			}
		case StateInDeclaration:
			{
				t.stateInDeclaration(c)
				break
			}
		case StateBeforeDeclaration:
			{
				t.stateBeforeDeclaration(c)
				break
			}
		case StateBeforeComment:
			{
				t.stateBeforeComment(c)
				break
			}
		case StateInProcessingInstruction:
			{
				t.stateInProcessingInstruction(c)
				break
			}
		case StateInEntity:
			{
				t.stateInEntity()
				break
			}
		}
		t.index += size
	}
	t.cleanup()
	t.finish()
}

/**
 * Remove data that has already been consumed from the buffer.
 */
func (t *Tokenizer) cleanup() {
	// If we are inside of text or attributes, emit what we already have.
	if t.sectionStart != t.index {
		if t.state == StateText || (t.state == StateInRCDATA && t.sequenceIndex == 0) {
			t.parser.ontext(t.sectionStart, t.index)
			t.sectionStart = t.index
		} else if t.state == StateInAttrValueDq || t.state == StateInAttrValueSq || t.state == StateInAttrValueNq {
			t.parser.onattribdata(t.sectionStart, t.index)
			t.sectionStart = t.index
		}
	}
}

func (t *Tokenizer) finish() {
	if t.state == StateInEntity {
		// TODO
		// t.entityDecoder!.end()
		t.state = t.baseState
	}

	t.handleTrailingData()

	t.parser.onend()
}

/** Handle any trailing data. */
func (t *Tokenizer) handleTrailingData() {
	endIndex := len(t.buffer)

	// If there is no remaining data, we are done.
	if t.sectionStart >= endIndex {
		return
	}

	if t.state == StateInCommentLike {
		if t.currentSequence == &SequenceCdataEnd {
			t.parser.oncdata(t.sectionStart, endIndex)
		} else {
			t.parser.oncomment(t.sectionStart, endIndex)
		}
	} else if t.state == StateInTagName ||
		t.state == StateBeforeAttrName ||
		t.state == StateBeforeAttrValue ||
		t.state == StateAfterAttrName ||
		t.state == StateInAttrName ||
		t.state == StateInDirName ||
		t.state == StateInDirArg ||
		t.state == StateInDirDynamicArg ||
		t.state == StateInDirModifier ||
		t.state == StateInAttrValueSq ||
		t.state == StateInAttrValueDq ||
		t.state == StateInAttrValueNq ||
		t.state == StateInClosingTagName {
		/*
		 * If we are currently in an opening or closing tag, us not calling the
		 * respective callback signals that the tag should be ignored.
		 */
	} else {
		t.parser.ontext(t.sectionStart, endIndex)
	}
}

// func (t *Tokenizer) emitCodePoint(cp int, consumed int) {
//   if t.baseState != StateText && t.baseState != StateInRCDATA {
//     if t.sectionStart < t.entityStart {
//       t.parser.onattribdata(t.sectionStart, t.entityStart)
//     }
//     t.sectionStart = t.entityStart + consumed
//     t.index = t.sectionStart - 1
//
//     t.parser.onattribentity(
//       fromCodePoint(cp),
//       t.entityStart,
//       t.sectionStart,
//     )
//   } else {
//     if t.sectionStart < t.entityStart {
//       t.parser.ontext(t.sectionStart, t.entityStart)
//     }
//     t.sectionStart = t.entityStart + consumed
//     t.index = t.sectionStart - 1
//
//     t.parser.ontextentity(
//       fromCodePoint(cp),
//       t.entityStart,
//       t.sectionStart,
//     )
//   }
//
// }
