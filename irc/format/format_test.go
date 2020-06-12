package ircf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var msgWithColor = "Hello, \x02Wor\x1dld\x0304,07\x1d!\x0f My name is \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"
var msgWithoutColor = "Hello, \x02Wor\x1dld\x1d!\x0f My name is \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"
var msgWithoutCodes = "Hello, World! My name is First Last. Testing resetONETWOTHREE. And reverse!"

func TestStrip(t *testing.T) {
	assert.Equal(t, msgWithoutColor, StripColor(msgWithColor))
	assert.Equal(t, msgWithoutCodes, StripCodes(msgWithColor))
}

func TestAllBlocks(t *testing.T) {
	expected := []Block{
		NewBlock("Hello, "),
		NewBlock("Wor", CharBold),
		NewBlock("ld", CharBold, CharItalics),
		NewColorBlock("!", 4, 7, CharBold),
		NewBlock(" My name is "),
		NewBlock("First", CharUnderline),
		NewBlock(" Last. Testing reset"),
		NewBlock("ONETWO", CharBold, CharItalics, CharUnderline, CharReverseColor),
		NewBlock("THREE. And "),
		NewBlock("reverse", CharReverseColor),
		NewBlock("!"),
	}

	assert.Equal(t, expected, Parse(msgWithColor))
}

func TestMarkdown(t *testing.T) {
	msgMarkdown := `Hello, **Wor*ld*!** My name is __First__ Last. Testing reset***__ONETWO__***THREE. And *reverse*!`
	assert.Equal(t,
		msgMarkdown,
		BlocksToMarkdown(Parse(msgWithoutColor)),
	)
	assert.Equal(t,
		msgMarkdown,
		BlocksToMarkdown(Parse(msgWithColor)),
	)
}

func TestMarkdownSpoilers(t *testing.T) {
	msgIRC := "In Game of Thrones, everyone\x031,1 dies\x03!"
	msgMarkdown := "In Game of Thrones, everyone|| dies||!"
	assert.Equal(t, msgMarkdown, BlocksToMarkdown(Parse(msgIRC)))
}
