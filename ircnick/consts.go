package ircnick

// https://github.com/lp0/charybdis/blob/imaginarynet/include/match.h#L92-L140

// add const (
const (
	// keep second word of each line
	// second line e.g: #define CNTRL_C   0x002
	//             becomes just CNTRL_C
	// add `= 1 << iota` to the end of the first line like below
	PRINT_C = 1 << iota
	CNTRL_C
	ALPHA_C
	PUNCT_C
	DIGIT_C
	SPACE_C
	NICK_C
	CHAN_C
	KWILD_C
	CHANPFX_C
	USER_C
	HOST_C
	NONEOS_C
	SERV_C
	EOL_C
	MWILD_C
	LET_C
	FCHAN_C
	FNICK_C

// add )
)
