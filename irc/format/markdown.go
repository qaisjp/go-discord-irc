package ircf

// From https://github.com/reactiflux/discord-irc/blob/87a3458bdde48290960405f2bf0cf53b7ff17b5e/lib/formatting.js#L25

func BlocksToMarkdown(blocks []Block) string {
	mdText := ""

	for i := 0; i < len(blocks)+1; i++ {
		// Default to unstyled blocks when index out of range
		block := Empty
		if i < len(blocks) {
			block = blocks[i]
		}
		prevBlock := Empty
		if i > 0 {
			prevBlock = blocks[i-1]
		}

		// Consider reverse as italic, some IRC clients use that
		prevItalic := prevBlock.Italic || prevBlock.Reverse
		italic := block.Italic || block.Reverse

		// If foreground == background, then spoiler
		prevSpoiler := prevBlock.Color != -1 && prevBlock.Color == prevBlock.Highlight
		spoiler := block.Color != -1 && block.Color == block.Highlight

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

		// NOTE: non-standard discord spoilers
		if !prevSpoiler && spoiler {
			mdText += "||"
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

		// NOTE: non-standard discord spoilers
		if prevSpoiler && !spoiler {
			mdText += "||"
		}

		mdText += block.Text
	}

	return mdText
}
