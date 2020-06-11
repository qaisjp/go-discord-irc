package ircf

import "fmt"

// From https://www.npmjs.com/package/irc-formatting 1.0.0-rc3

type Block struct {
	Bold, Italic, Underline, Reverse bool
	Color, Highlight                 int
	Text                             string
}

var Empty = NewBlock("")

func NewBlock(text string, fields ...rune) (this Block) {
	this.Text = text
	this.Color = -1
	this.Highlight = -1

	for _, code := range fields {
		this.SetField(code, true)
	}

	return
}

func (this Block) Extend(text string) (ret Block) {
	ret = this
	ret.Text = text
	if ret.Color > 99 {
		ret.Color = 99
	}

	if ret.Highlight > 99 {
		ret.Highlight = 99
	}
	return
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

func (this *Block) codeToField(code rune) (field *bool) {
	if code == CharacterBold {
		field = &this.Bold
	} else if code == CharacterItalics {
		field = &this.Italic
	} else if code == CharacterUnderline {
		field = &this.Underline
	} else if code == CharacterReverseColor {
		field = &this.Reverse
	}
	return field
}

func (this *Block) SetField(code rune, val bool) {
	if field := this.codeToField(code); field != nil {
		*field = val
		return
	}
	panic(fmt.Sprintf(`Unknown code \x%x`, code))
}

func (this Block) GetField(code rune) bool {
	if field := this.codeToField(code); field != nil {
		return *field
	}
	panic(fmt.Sprintf(`Unknown code \x%x`, code))
}
