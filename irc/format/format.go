package ircf

import (
	"regexp"
)

// This file is a subset of https://www.npmjs.com/package/irc-formatting 1.0.0-rc3

// Characters includes all the codes defined in https://modern.ircdocs.horse/formatting.html
const (
	CharacterBold          rune = '\x02'
	CharacterItalics            = '\x1D'
	CharacterUnderline          = '\x1F'
	CharacterStrikethrough      = '\x1E'
	CharacterMonospace          = '\x11'
	CharacterColor              = '\x03'
	CharacterHex                = '\x04'
	CharacterReverseColor       = '\x16'
	CharacterReset              = '\x0F'
)

var colorRegex = regexp.MustCompile(`\x03(\d\d?)(,(\d\d?))?/g`)
var colorRegexStrip = regexp.MustCompile(`\x03\d{0,2}(,\d{0,2}|\x02\x02)?`)

var Keys = map[rune]string{
	CharacterBold:      "bold",
	CharacterItalics:   "italic",
	CharacterUnderline: "underline",
}

const TagBold = "b"
const TagItalic = "i"
const TagUnderline = "u"
const TagBlock = "span"
const TagLine = "p"

const ClassReverse = "ircf-reverse"
const ClassColorPref = "ircf-fg-"
const ClassHighlightPref = "ircf-bg-"
const ClassNoColor = "ircf-no-color"
const ClassLine = "ircf-line"

func StripColor(text string) string {
	return colorRegexStrip.ReplaceAllString(text, "")
}

func Parse(text string) (result []Block) {
	result = []Block{}
	prev := NewBlock("")
	startIndex := 0

	// Append a resetter to simplify code a bit
	text += string(CharacterReset)

	for i, ch := range text {
		var current Block
		updated := true
		skip := 0
		nextStart := -1

		switch ch {
		// bold, italic, underline
		case CharacterBold:
			fallthrough
		case CharacterItalics:
			fallthrough
		case CharacterUnderline:
			current = prev.Extend("")

			// Toggle style
			current.SetField(ch, !prev.GetField(ch))

		// color
		case CharacterColor:
			panic("Colors not supported")

		// reverse
		case CharacterReverseColor:
			current = prev.Extend("")

			if prev.Color != -1 {
				current.Color = prev.Highlight
				current.Highlight = prev.Color

				if current.Color == -1 {
					current.Color = 0
				}
			}

			current.Reverse = !prev.Reverse

		// reset
		case CharacterReset:
			current = NewBlock("")

		default:
			updated = false
		}

		if updated {
			prev.Text = text[startIndex:i]

			if nextStart != -1 {
				startIndex = nextStart
			} else {
				startIndex = i + 1
			}

			if len(prev.Text) > 0 {
				result = append(result, prev)
			}

			prev = current
		}

		i += skip
	}

	return result
}
