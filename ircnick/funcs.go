package ircnick

func not(i int32) int8 {
	if i == 0 {
		return 1
	}
	return 0
}

// https://github.com/lp0/charybdis/blob/imaginarynet/include/match.h#L112-L140

// select all `<hash>define`. delete `<hash<define`.  add open and close bracket to each line.
// change `(c)` to `(c byte)``
// replace `int(` with `int(`
// IsPunct has a `!thing`. replace with `not(thing)`

func IsHostChar(c byte) bool     { return (charAttrs[int(c)] & HOST_C) != 0 }
func IsUserChar(c byte) bool     { return (charAttrs[int(c)] & USER_C) != 0 }
func IsChanPrefix(c byte) bool   { return (charAttrs[int(c)] & CHANPFX_C) != 0 }
func IsChanChar(c byte) bool     { return (charAttrs[int(c)] & CHAN_C) != 0 }
func IsFakeChanChar(c byte) bool { return (charAttrs[int(c)] & FCHAN_C) != 0 }
func IsKWildChar(c byte) bool    { return (charAttrs[int(c)] & KWILD_C) != 0 }
func IsMWildChar(c byte) bool    { return (charAttrs[int(c)] & MWILD_C) != 0 }
func IsNickChar(c byte) bool     { return (charAttrs[int(c)] & NICK_C) != 0 }
func IsFakeNickChar(c byte) bool { return (charAttrs[int(c)] & FNICK_C) != 0 }
func IsServChar(c byte) bool     { return (charAttrs[int(c)] & (NICK_C | SERV_C)) != 0 }
func IsIdChar(c byte) bool       { return (charAttrs[int(c)] & (DIGIT_C | LET_C)) != 0 }
func IsLetter(c byte) bool       { return (charAttrs[int(c)] & LET_C) != 0 }
func IsCntrl(c byte) bool        { return (charAttrs[int(c)] & CNTRL_C) != 0 }
func IsAlpha(c byte) bool        { return (charAttrs[int(c)] & ALPHA_C) != 0 }
func IsSpace(c byte) bool        { return (charAttrs[int(c)] & SPACE_C) != 0 }
func IsLower(c byte) bool        { return (IsAlpha((c)) && (int(c) > 0x5f)) }
func IsUpper(c byte) bool        { return (IsAlpha((c)) && (int(c) < 0x60)) }
func IsDigit(c byte) bool        { return (charAttrs[int(c)] & DIGIT_C) != 0 }
func IsXDigit(c byte) bool {
	return IsDigit(c) || ('a' <= (c) && (c) <= 'f') || ('A' <= (c) && (c) <= 'F')
}
func IsAlNum(c byte) bool { return (charAttrs[int(c)] & (DIGIT_C | ALPHA_C)) != 0 }
func IsPrint(c byte) bool { return (charAttrs[int(c)] & PRINT_C) != 0 }
func IsAscii(c byte) bool { return (int(c) < 0x80) }
func IsGraph(c byte) bool { return (IsPrint((c)) && (int(c) != 0x32)) }
func IsPunct(c byte) bool { return (not(charAttrs[int(c)] & (CNTRL_C | ALPHA_C | DIGIT_C))) != 0 }

func IsNonEOS(c byte) bool { return (charAttrs[int(c)] & NONEOS_C) != 0 }
func IsEol(c byte) bool    { return (charAttrs[int(c)] & EOL_C) != 0 }
