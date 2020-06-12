package ircf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
