package utils

import (
	"runtime"
	"strings"
	"unicode"
)

func split(src string) []string {
	var (
		entries   []string
		runes     [][]rune
		lastClass int
		class     int
	)
	for _, r := range src {
		switch {
		case unicode.IsLower(r):
			class = 1
		case unicode.IsUpper(r):
			class = 2
		case unicode.IsDigit(r):
			class = 3
		default:
			class = 4
		}
		if class == lastClass || class == 3 {
			runes[len(runes)-1] = append(runes[len(runes)-1], r)
		} else {
			runes = append(runes, []rune{r})
		}
		lastClass = class
	}
	// handle upper case -> lower case sequences, e.g.
	// "PDFL", "oader" -> "PDF", "Loader"
	for i := 0; i < len(runes)-1; i++ {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}
	for _, s := range runes {
		if len(s) > 0 {
			entries = append(entries, string(s))
		}
	}
	return entries
}

func ToSnakeCase(str string) string {
	return strings.Join(split(str), "_")
}

func GetOriginalCallerFuncName(skip int) string {
	pc := make([]uintptr, skip+1) // at least 1 entry needed
	runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:3])
	frame, _ := frames.Next()
	return frame.Function
}

func GetOpNameBySnakeCase(caller string) string {
	callerPath := strings.Split(caller, ".")
	return strings.ToLower(
		strings.Join(
			split(
				callerPath[len(callerPath)-1]), "_",
		),
	)
}

func GetOpName(caller string) string {
	callerPath := strings.Split(caller, ".")
	return strings.ToLower(
		strings.Join(
			split(
				callerPath[len(callerPath)-1]), " ",
		),
	)
}
