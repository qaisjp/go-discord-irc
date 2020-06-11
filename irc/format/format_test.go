package ircf

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var colorRegexRepl = regexp.MustCompile(`\x03\d{0,2}(,\d{0,2}|\x02\x02)?`)
var msgWithColor = "Hello, \u0002Wor\x1dld\u000304,07\x1d!\u000f My name is \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"
var msgWithoutColor = "Hello, \u0002Wor\x1dld\x1d!\u000f My name is \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"
var msgWithoutCodes = "Hello, World! My name is First Last. Testing resetONETWOTHREE. And reverse!"

func TestStrip(t *testing.T) {
	assert.Equal(t, msgWithoutColor, StripColor(msgWithColor))
	assert.Equal(t, msgWithoutCodes, StripCodes(msgWithColor))
}

func TestAllBlocks(t *testing.T) {
	msg := msgWithoutColor

	// fmt.Println("\nMarkdown:")
	// fmt.Println(IRCToMarkdown(msg))

	expected := []Block{
		NewBlock("Hello, "),
		NewBlock("Wor", CharBold),
		NewBlock("ld", CharBold, CharItalics),
		NewBlock("!", CharBold),
		NewBlock(" My name is "),
		NewBlock("First", CharUnderline),
		NewBlock(" Last. Testing reset"),
		NewBlock("ONETWO", CharBold, CharItalics, CharUnderline, CharReverseColor),
		NewBlock("THREE. And "),
		NewBlock("reverse", CharReverseColor),
		NewBlock("!"),
	}

	assert.Equal(t, expected, Parse(msg))
}

func TestMarkdown(t *testing.T) {
	assert.Equal(t,
		`Hello, **Wor*ld*!** My name is __First__ Last. Testing reset***__ONETWO__***THREE. And *reverse*!`,
		IRCToMarkdown(msgWithoutColor),
	)
}

/*
Blocks:

{Bold:false Italic:false Underline:false Reverse:false Color:-1 Highlight:-1 Text:Hello, }
{Bold:true Italic:false Underline:false Reverse:false Color:-1 Highlight:-1 Text:Wor}
{Bold:true Italic:true Underline:false Reverse:false Color:-1 Highlight:-1 Text:ld}
{Bold:true Italic:false Underline:false Reverse:false Color:-1 Highlight:-1 Text:!}
{Bold:false Italic:false Underline:false Reverse:false Color:-1 Highlight:-1 Text: My name is }
{Bold:false Italic:false Underline:true Reverse:false Color:-1 Highlight:-1 Text:First}
{Bold:false Italic:false Underline:false Reverse:false Color:-1 Highlight:-1 Text: Last. Testing reset}
{Bold:true Italic:true Underline:true Reverse:true Color:-1 Highlight:-1 Text:ONETWO}
{Bold:false Italic:false Underline:false Reverse:false Color:-1 Highlight:-1 Text:THREE. And }
{Bold:false Italic:false Underline:false Reverse:true Color:-1 Highlight:-1 Text:reverse}
{Bold:false Italic:false Underline:false Reverse:false Color:-1 Highlight:-1 Text:!}


*/
