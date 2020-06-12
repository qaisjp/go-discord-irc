package ircf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var msgWithColor = "Hello, \x02Wor\x1dld\x0304,07\x1d! \x1dMy name is\x1d\x0f... \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"
var msgWithoutColor = "Hello, \x02Wor\x1dld\x1d! \x1dMy name is\x1d\x0f... \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"
var msgWithoutCodes = "Hello, World! My name is... First Last. Testing resetONETWOTHREE. And reverse!"

func TestStrip(t *testing.T) {
	assert.Equal(t, msgWithoutColor, StripColor(msgWithColor))
	assert.Equal(t, msgWithoutCodes, StripCodes(msgWithColor))
}

func TestAllBlocks(t *testing.T) {
	expected := []Block{
		NewBlock("Hello, "),
		NewBlock("Wor", CharBold),
		NewBlock("ld", CharBold, CharItalics),
		NewColorBlock("! ", 4, 7, CharBold),
		NewColorBlock("My name is", 4, 7, CharBold, CharItalics),
		NewBlock("... "),
		NewBlock("First", CharUnderline),
		NewBlock(" Last. Testing reset"),
		NewBlock("ONETWO", CharBold, CharItalics, CharUnderline, CharReverseColor),
		NewBlock("THREE. And "),
		NewBlock("reverse", CharReverseColor),
		NewBlock("!"),
	}

	assert.Equal(t, expected, Parse(msgWithColor))
}

func TestFormatSimple(t *testing.T) {
	cases := []struct {
		Message  string
		Input    string
		Expected string
	}{
		{"spoilers", "\x0306,06text\x03", "||text||"},

		// Test against https://github.com/reactiflux/discord-irc/blob/41f8444d17b2b282c437b5f871f0252968c03525/test/formatting.test.js
		{"bold", "\x02text\x02", "**text**"},
		{"reverse", "\x16text\x16", "*text*"},
		{"italics", "\x1dtext\x1d", "*text*"},
		{"underline", "\x1ftext\x1f", "__text__"},
		{"color", "\x0306,08text\x03", "text"},
		{"bold with nested italics", "\x02bold \x16italics\x16\x02", "**bold *italics***"},
		{"bold with nested underline", "\x02bold \x1funderline\x1f\x02", "**bold __underline__**"},
	}

	for _, c := range cases {
		t.Run(c.Message, func(t *testing.T) {
			assert.Equal(t, c.Expected, BlocksToMarkdown(Parse(c.Input)))
		})
	}
}

func TestMarkdown(t *testing.T) {
	msgMarkdown := `Hello, **Wor*ld*! *My name is***... __First__ Last. Testing reset***__ONETWO__***THREE. And *reverse*!`
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
