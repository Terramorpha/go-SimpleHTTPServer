package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

const ( //terminal colors
	//ColorReset resets all colors/properties in terminal escape sequences
	ColorReset = 0
	//ColorBold makes text bold
	ColorBold       = 1
	ColorDim        = 2
	ColorUnderlined = 4
	ColorBlink      = 5
	ColorReversed   = 7
	ColorHidden     = 8

	ColorBlack   = 0
	ColorRed     = 1
	ColorGreen   = 2
	ColorYellow  = 3
	ColorBlue    = 4
	ColorMagenta = 5
	ColorCyan    = 6
	ColorGrey    = 7
)

func Fprint(w io.Writer, x ...interface{}) (int, error) {
	if MainConfig.Get("IsTellTime").Bool() {
		return fmt.Fprint(w, time.Now().Format("Jan 2 15:04:05 MST 2006"), fmt.Sprint(x...))
	}
	return fmt.Fprint(w, x...)
}
func TextColor(colorCode int) string {
	return fmt.Sprintf("\033[3%dm", colorCode)
}

func TextStyle(styleCode int) string {
	return fmt.Sprintf("\033[%dm", styleCode)
}

func TextReset() string {
	return fmt.Sprintf("\033[%dm", ColorReset)
}

func Fatal(x interface{}) {
	ePrintln(x)
	os.Exit(1)
	//panic(x)not pretty to the user lol
}

func iPrint(a ...interface{}) (int, error) {
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[INFOS]", ColorGreen), fmt.Sprint(a...)))
}

func iPrintln(a ...interface{}) (int, error) {
	return iPrint(fmt.Sprintln(a...))
}

func iPrintf(format string, a ...interface{}) (int, error) {
	return iPrint(fmt.Sprintf(format, a...))
}

func ePrint(a ...interface{}) (int, error) {
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[ERROR]", ColorRed), fmt.Sprint(a...)))
}

func ePrintln(a ...interface{}) (int, error) {
	return ePrint(fmt.Sprintln(a...))
}

func ePrintf(format string, a ...interface{}) (int, error) {
	return ePrint(fmt.Sprintf(format, a...))
}

func wPrint(a ...interface{}) (int, error) {
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[WARN ]", ColorYellow), fmt.Sprint(a...)))
}

func wPrintln(a ...interface{}) (int, error) {
	return wPrint(fmt.Sprintln(a...))
}

func wPrintf(format string, a ...interface{}) (int, error) {
	return wPrint(fmt.Sprintf(format, a...))
}

func vPrint(verbosityTreshold int, x ...interface{}) (int, error) {
	if MainConfig.Get("Verbosity").Int() < verbosityTreshold {
		return 0, nil
	}

	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[VERBO]", ColorGrey), fmt.Sprint(x...)))

}

func vPrintf(verbosityTreshold int, f string, a ...interface{}) (int, error) { //log des choses selon le degré de verbosité

	return vPrint(verbosityTreshold, fmt.Sprintf(f, a...))
}

func vPrintln(t int, a ...interface{}) (int, error) {
	return vPrint(t, a...)
}

func Colorize(x string, code int) string {
	if MainConfig.Get("IsColored").Bool() {
		return TextColor(code) + x + TextReset()
	}
	return x
}

func dPrintln(x ...interface{}) (int, error) {
	if !MainConfig.Get("IsDebug").Bool() {
		return 0, nil
	}
	return dPrint(fmt.Sprintln(x...))
}

func dPrint(x ...interface{}) (int, error) {

	if !MainConfig.Get("IsDebug").Bool() {
		return 0, nil
	}
	return Fprint(os.Stderr, fmt.Sprintf("%s %s", Colorize("[DEBUG]", ColorCyan), fmt.Sprint(x...)))
}

func dPrintf(format string, x ...interface{}) (int, error) {
	if !MainConfig.Get("IsDebug").Bool() {
		return 0, nil
	}
	return dPrint(fmt.Sprintf(format, x...))
}
