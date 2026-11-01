package main

import "strings"

// Replace leading indentation tabs throughout the file. Token offsets protect
// multiline string contents; tabs after the first non-whitespace byte stay literal.
func normalizeIndentation(src string) (string, error) {
	tokens, err := tokenize(src)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.Grow(len(src))
	tokenIndex := 0
	for start := 0; start < len(src); {
		end := len(src)
		if n := strings.IndexByte(src[start:], '\n'); n >= 0 {
			end = start + n + 1
		}
		for tokenIndex < len(tokens) && tokens[tokenIndex].end <= start {
			tokenIndex++
		}
		literal := tokenIndex < len(tokens) && tokens[tokenIndex].quoted && tokens[tokenIndex].start < start && tokens[tokenIndex].end > start
		if literal {
			out.WriteString(src[start:end])
		} else {
			pos := start
			for pos < end && (src[pos] == ' ' || src[pos] == '\t') {
				if src[pos] == '\t' {
					out.WriteString("    ")
				} else {
					out.WriteByte(' ')
				}
				pos++
			}
			out.WriteString(src[pos:end])
		}
		start = end
	}
	return out.String(), nil
}
