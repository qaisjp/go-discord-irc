package ircf

// From https://github.com/reactiflux/discord-irc/blob/87a3458bdde48290960405f2bf0cf53b7ff17b5e/lib/formatting.js#L25

func BlocksToMarkdown(blocks []Block) string {
	mdText := ""

	for i, block := range blocks {
		// Default to unstyled blocks when index out of range
		prevBlock := Empty
		if i > 0 {
			prevBlock = blocks[i-1]
		}

		// Consider reverse as italic, some IRC clients use that
		prevItalic := prevBlock.Italic || prevBlock.Reverse
		italic := block.Italic || block.Reverse

		// Add start markers when style turns from false to true
		if !prevItalic && italic {
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
		if prevItalic && !italic {
			mdText += "*"
		}

		mdText += block.Text
	}

	return mdText
}
