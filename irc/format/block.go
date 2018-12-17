package ircf

import "fmt"

// From https://www.npmjs.com/package/irc-formatting 1.0.0-rc3

type Block struct {
	Bold, Italic, Underline, Reverse bool
	Color, Highlight                 int
	Text                             string
}

var Empty = NewBlock(nil, "")

func NewBlock(prev *Block, text string) *Block {
	this := &Block{}
	this.Color = -1
	this.Highlight = -1
	this.Text = text

	if prev != nil {
		this.Bold = prev.Bold
		this.Italic = prev.Italic
		this.Underline = prev.Underline
		this.Reverse = prev.Reverse
		this.Color = prev.Color
		this.Highlight = prev.Highlight
	}

	if this.Color > 99 {
		this.Color = 99
	}

	if this.Highlight > 99 {
		this.Highlight = 99
	}

	return this
}

func (this Block) Equals(other Block) bool {
	return this.Bold == other.Bold &&
		this.Italic == other.Italic &&
		this.Underline == other.Underline &&
		this.Reverse == other.Reverse &&
		this.Color == other.Color &&
		this.Highlight == other.Highlight
}

func (this Block) IsPlain() bool {
	return (!this.Bold && !this.Italic && !this.Underline && !this.Reverse &&
		this.Color == -1 && this.Highlight == -1)
}

func (this Block) HasSameColor(other Block, reversed bool) bool {
	if this.Reverse && reversed {
		return ((this.Color == other.Highlight || other.Highlight == -1) && this.Highlight == other.Color)
	}
	return (this.Color == other.Color && this.Highlight == other.Highlight)
}

func (this Block) GetColorString() string {
	var str = ""

	if this.Color != -1 {

		str = fmt.Sprintf("%02d", this.Color)
	}

	if this.Highlight != -1 {
		str += "," + fmt.Sprintf("%02d", this.Highlight)
	}

	return str
}

func (this *Block) SetField(code string, val bool) {
	if code == B {
		this.Bold = val
	} else if code == I {
		this.Italic = val
	} else if code == U {
		this.Underline = val
	} else {
		panic("Unknown code " + code)
	}
}

func (this Block) GetField(code string) bool {
	if code == B {
		return this.Bold
	} else if code == I {
		return this.Italic
	} else if code == U {
		return this.Underline
	} else {
		panic("Unknown code " + code)
	}
}
