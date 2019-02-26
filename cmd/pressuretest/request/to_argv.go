package request

import (
	"fmt"
	"runtime"
)

type UnmatchedQuota struct{}

func (e *UnmatchedQuota) Error() string {
	return fmt.Sprintf("unmatched quota")
}

// from https://github.com/mgutz/str/blob/master/funcsPZ.go#L275
func ToArgv(s string) ([]string, error) {
	const (
		InArg = iota
		InQuotedArg
		OutArg
	)
	currentState := OutArg
	currentQuoteChar := "\x00" // to distinguish between ' and " quotations
	currentArg := ""
	var argv []string

	isQuote := func(c string) bool {
		return c == `"` || c == `'`
	}

	isEscape := func(c string) bool {
		return c == `\`
	}

	isWhitespace := func(c string) bool {
		return c == " " || c == "\t"
	}

	L := len(s)
	for i := 0; i < L; i++ {
		c := s[i : i+1]

		if isQuote(c) {
			switch currentState {
			case OutArg:
				currentArg = ""
				fallthrough
			case InArg:
				currentState = InQuotedArg
				currentQuoteChar = c

			case InQuotedArg:
				if c == currentQuoteChar {
					currentState = InArg
				} else {
					currentArg += c
				}
			}

		} else if isWhitespace(c) {
			switch currentState {
			case InArg:
				argv = append(argv, currentArg)
				currentState = OutArg
			case InQuotedArg:
				currentArg += c
			case OutArg:
				// nothing
			}

		} else if isEscape(c) {
			switch currentState {
			case OutArg:
				currentArg = ""
				currentState = InArg
				fallthrough
			case InArg:
				fallthrough
			case InQuotedArg:
				if i == L-1 {
					if runtime.GOOS == "windows" {
						// just add \ to end for windows
						currentArg += c
					} else {
						panic("Escape character at end string")
					}
				} else {
					if runtime.GOOS == "windows" {
						peek := s[i+1 : i+2]
						if peek != `"` {
							currentArg += c
						}
					} else {
						i++
						c = s[i : i+1]
						currentArg += c
					}
				}
			}
		} else {
			switch currentState {
			case InArg, InQuotedArg:
				currentArg += c

			case OutArg:
				currentArg = ""
				currentArg += c
				currentState = InArg
			}
		}
	}

	if currentState == InArg {
		argv = append(argv, currentArg)
	} else if currentState == InQuotedArg {
		return argv, &UnmatchedQuota{}
	}

	return argv, nil
}
