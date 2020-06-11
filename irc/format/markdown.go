package ircf

// From https://github.com/reactiflux/discord-irc/blob/87a3458bdde48290960405f2bf0cf53b7ff17b5e/lib/formatting.js#L25

func IRCToMarkdown(text string) string {
	blocks := Parse(text)
	for i, b := range blocks {
		// Consider reverse as italic, some IRC clients use that
		if b.Reverse {
			blocks[i].Italic = true
		}
	}

	mdText := ""

	for i := 0; i <= len(blocks); i++ {
		// Default to unstyled blocks when index out of range
		var block Block
		if i < len(blocks) {
			block = blocks[i]
		}
		var prevBlock Block
		if i > 0 {
			prevBlock = blocks[i-1]
		}

		// Add start markers when style turns from false to true
		if !prevBlock.Italic && block.Italic {
			mdText += "*"
		}
		if !prevBlock.Bold && block.Bold {
			mdText += "**"
		}
		if !prevBlock.Underline && block.Underline {
			mdText += "__"
		}

		// Add end markers when style turns from true to false
		// (and apply in reverse order to maintain nesting)
		if prevBlock.Underline && !block.Underline {
			mdText += "__"
		}
		if prevBlock.Bold && !block.Bold {
			mdText += "**"
		}
		if prevBlock.Italic && !block.Italic {
			mdText += "*"
		}

		mdText += block.Text
	}

	return mdText
}
