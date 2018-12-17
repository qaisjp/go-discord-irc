package ircf

import (
	"regexp"
)

// Subset of https://www.npmjs.com/package/irc-formatting 1.0.0-rc3

const B = "\x02"
const I = "\x1d"
const U = "\x1f"
const C = "\x03"
const R = "\x16"
const O = "\x0f"

var colorRegex = regexp.MustCompile(`\x03(\d\d?)(,(\d\d?))?/g`)

var Keys = map[string]string{
	"\x02": "bold",
	"\x1d": "italic",
	"\x1f": "underline",
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

func Parse(text string) (result []*Block) {
	result = []*Block{}
	current := NewBlock(nil, "")
	startIndex := 0

	// Append a resetter to simplify code a bit
	text += R

	for i := 0; i < len(text); i++ {
		ch := text[i]
		var prev *Block
		skip := 0
		nextStart := -1

		switch ch {
		// bold, italic, underline
		case '\x02':
			fallthrough
		case '\x1d':
			fallthrough
		case '\x1f':
			{
				prev = current
				current = NewBlock(prev, "")

				// Toggle style
				current.SetField(string(ch), !prev.GetField(string(ch)))
			}

		// color
		case '\x03':
			{
				panic("Colors not supported")
			}

		// reverse
		case '\x16':
			{
				prev = current
				current = NewBlock(prev, "")

				if prev.Color != -1 {
					current.Color = prev.Highlight
					current.Highlight = prev.Color

					if current.Color == -1 {
						current.Color = 0
					}
				}

				current.Reverse = !prev.Reverse
			}

		// reset
		case '\x0f':
			{
				prev = current
				current = NewBlock(nil, "")
			}
		}

		if prev != nil {
			prev.Text = text[startIndex:i]

			if nextStart != -1 {
				startIndex = nextStart
			} else {
				startIndex = i + 1
			}

			if len(prev.Text) > 0 {
				result = append(result, prev)
			}
		}

		i += skip
	}

	return result
}
