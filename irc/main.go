package main

import (
	"fmt"
	"regexp"

	"github.com/qaisjp/go-discord-irc/irc/format"
)

var colorRegexRepl = regexp.MustCompile(`\x03\d{0,2}(,\d{0,2}|\x02\x02)?`)
var msg = "Hello, \u0002Wor\x1dld\u000304,07\x1d!\u000f My name is \x1fqais\x1f patankar. Testing reset\x1f\x1d\x02\x16ONETWO\x0fTHREE. And \x16reverse\x16!"

func main() {
	stripped := colorRegexRepl.ReplaceAllString(msg, "")
	fmt.Println("Blocks:\n")
	for _, block := range ircf.Parse(stripped) {
		fmt.Printf("%+v\n", *block)
	}

	fmt.Println("\nMarkdown:\n")
	fmt.Println(ircf.IRCToMarkdown(stripped))
}
