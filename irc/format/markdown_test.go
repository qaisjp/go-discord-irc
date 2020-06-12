package ircf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownSimple(t *testing.T) {
	cases := []struct {
		Message  string
		Input    string
		Expected string
	}{
		{"spoilers", "\x0306,06text\x03", "||text||"},

		// Test against https://github.com/reactiflux/discord-irc/blob/41f8444d17b2b282c437b5f871f0252968c03525/test/formatting.test.js
		{"bold", "\x02text\x02", "**text**"},
		{"reverse", "\x16text\x16", "*text*"},
		{"italics", "\x1dtext\x1d", "*text*"},
		{"underline", "\x1ftext\x1f", "__text__"},
		{"color", "\x0306,08text\x03", "text"},
		{"bold with nested italics", "\x02bold \x16italics\x16\x02", "**bold *italics***"},
		{"bold with nested underline", "\x02bold \x1funderline\x1f\x02", "**bold __underline__**"},
	}

	for _, c := range cases {
		t.Run(c.Message, func(t *testing.T) {
			assert.Equal(t, c.Expected, BlocksToMarkdown(Parse(c.Input)))
		})
	}
}

func TestMarkdownLong(t *testing.T) {
	msgMarkdown := `Hello, **Wor*ld*! *My name is***... __First__ Last. Testing reset***__ONETWO__***THREE. And *reverse*!`
	assert.Equal(t,
		msgMarkdown,
		BlocksToMarkdown(Parse(msg)),
	)
}

func TestMarkdownSpoilers(t *testing.T) {
	msgIRC := "In Game of Thrones, everyone\x031,1 dies\x03!"
	msgMarkdown := "In Game of Thrones, everyone|| dies||!"
	assert.Equal(t, msgMarkdown, BlocksToMarkdown(Parse(msgIRC)))
}
