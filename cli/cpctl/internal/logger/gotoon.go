package logger

import (
	"fmt"

	"github.com/alpkeskin/gotoon"
)

// Message prints a neutral informational message for humans.
func Message(message string) {
	printStyled("info", message)
}

// Warning prints a warning message for humans.
func Warning(message string) {
	printStyled("warn", message)
}

// Failure prints an error message for humans.
func Failure(message string) {
	printStyled("error", message)
}

// --- internal helpers ---

func printStyled(level, message string) {
	encoded, err := gotoon.Encode(map[string]string{
		"level": level,
		"msg":   message,
	})
	if err != nil {
		// Fallback: never fail logging because of UI
		fmt.Println(message)
		return
	}

	fmt.Println(encoded)
}
