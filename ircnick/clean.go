package ircnick

// NickClean replaces invalid characters with an underscore
func NickClean(nick string) string {
	// https://github.com/lp0/charybdis/blob/9ced2a7932dddd069636fe6fe8e9faa6db904703/ircd/client.c#L854-L884
	if nick[0] == '-' {
		nick = "_" + nick
	}
	if IsDigit(nick[0]) {
		nick = "_" + nick
	}

	newNick := []byte(nick)

	// Replace bad characters with underscores
	for i, c := range []byte(nick) {
		if !IsNickChar(c) || IsFakeNickChar(c) {
			newNick[i] = '_'
		} else {
			// newNick[i] = c
		}
	}

	return string(newNick)
}
