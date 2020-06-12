package ircf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var msg = "Hello, \x02Wor\x1dld\x0304,07\x1d! \x1dMy name is\x1d\x0f... \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"

func TestStrip(t *testing.T) {
	msgStripped := "Hello, World! My name is... First Last. Testing resetONETWOTHREE. And reverse!"
	assert.Equal(t, msgStripped, StripCodes(msg))
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

	assert.Equal(t, expected, Parse(msg))
}
