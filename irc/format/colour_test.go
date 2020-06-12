package ircf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripColor(t *testing.T) {
	msgStripped := "Hello, \x02Wor\x1dld\x1d! \x1dMy name is\x1d\x0f... \x1fFirst\x1f Last. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"
	assert.Equal(t, msgStripped, StripColor(msg))
}

func TestColorReset(t *testing.T) {
	assert.Equal(t,
		[]Block{
			NewBlock("Nothing"),
			NewColorBlock("Coloured", 5, 6),
			NewBlock("ResetColor"),
			NewColorBlock("Suffix", 3, 4),
		},
		Parse("Nothing"+
			"\x035,6Coloured"+
			"\x03ResetColor"+
			"\x033,4Suffix"),
	)
}

func TestColorSimple(t *testing.T) {
	assert.Equal(t,
		[]Block{
			NewBlock("Hello "),
			NewColorBlock("everyone", 4, -1),
		},
		Parse("Hello \x034everyone"),
	)
}
