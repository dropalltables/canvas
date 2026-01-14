package ui

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	cyan    = color.New(color.FgCyan)
	yellow  = color.New(color.FgYellow)
	green   = color.New(color.FgGreen)
	red     = color.New(color.FgRed)
	bold    = color.New(color.Bold)
	faint   = color.New(color.Faint)
)

func Question(format string, a ...interface{}) {
	cyan.Print("? ")
	fmt.Printf(format+"\n", a...)
}

func Warning(format string, a ...interface{}) {
	yellow.Print("! ")
	fmt.Printf(format+"\n", a...)
}

func Info(format string, a ...interface{}) {
	green.Print("* ")
	fmt.Printf(format+"\n", a...)
}

func Error(format string, a ...interface{}) {
	red.Print("X ")
	fmt.Printf(format+"\n", a...)
}

func Bold(s string) string {
	return bold.Sprint(s)
}

func Faint(s string) string {
	return faint.Sprint(s)
}

func Green(s string) string {
	return green.Sprint(s)
}

func Yellow(s string) string {
	return yellow.Sprint(s)
}

func Red(s string) string {
	return red.Sprint(s)
}

func Cyan(s string) string {
	return cyan.Sprint(s)
}
