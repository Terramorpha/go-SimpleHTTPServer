package main

import (
	"io"
	"runtime"
	"strconv"
	"strings"
)

func ParseCommandLine(s string) []string {
	var (
		outputSlice = make([]string, 0)
		currentWord []rune

		doubleQuoted bool
		quoted       bool
		escaped      bool
	)
	currentWord = make([]rune, 0, 32)
	for _, char := range s { //ranging through each char
		//fmt.Printf("char: %c\n", char)
		//fmt.Println(string(currentWord))
		if escaped { // if char before was \ :
			currentWord = append(currentWord, char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '\'' {
			quoted = !quoted
			continue
		}

		if char == '"' {
			doubleQuoted = !doubleQuoted
			continue
		}

		if char == ' ' {
			outputSlice = append(outputSlice, string(currentWord))
			currentWord = make([]rune, 0, 32)
			//fmt.Println(outputSlice)
			continue
		}
		currentWord = append(currentWord, char)

	}
	if len(currentWord) > 0 {
		outputSlice = append(outputSlice, string(currentWord))
	}
	outputSlice = StripBlankStrings(outputSlice)

	return outputSlice
}

func StripBlankStrings(s []string) []string {
	o := make([]string, len(s))
	for i := range s {
		if len(s[i]) == 0 {
			continue
		}
		o = append(o, s[i])
	}
	return o
}

func Extension(s string) string {
	a := strings.Split(s, ".")
	return a[len(a)-1]
}

func use(x ...interface{}) {

}
func ReadSlice(r io.Reader, delim []byte) ([]byte, error) {
	var (
		iDelim    int
		out       []byte = make([]byte, 0, len(delim))
		middleMan []byte = make([]byte, 0)
		oneByte          = make([]byte, 1)
	)

	for iDelim < len(delim) {
		//dPrintln("i", iDelim, string(out))
		_, err := r.Read(oneByte)
		if err != nil {
			return out, err
		}
		if oneByte[0] != delim[iDelim] {
			out = append(out, middleMan...)
			middleMan = make([]byte, 0)
			out = append(out, oneByte[0])
			iDelim = 0
			continue
		}
		middleMan = append(middleMan, oneByte[0])
		iDelim++
	}
	return out, nil
}

func Line(skip ...int) string { //tells line number
	var s int
	if len(skip) == 0 {
		s = 1
	} else {

		s = skip[0]
	}
	_, file, a, _ := runtime.Caller(s)
	split := strings.Split(file, "/")
	file = split[len(split)-1]
	return file + " " + strconv.Itoa(a)
}

var (
	StringValueBool = []string{
		"true",
		"false",
		"True",
		"False",
		"TRUE",
		"FALSE",
		"1",
		"0",
		"on",
		"off",
		"ON",
		"OFF",
	}
)

func StringInArray(x string, y []string) bool {
	for i := range y {
		if x == y[i] {
			return true
		}
	}
	return false
}

func CheckBool(x string) (bool, error) {
	b, err := strconv.ParseBool(x)
	return b, err
}

func CheckInt(x string) (int, error) {
	b, err := strconv.Atoi(x)
	return b, err
}
