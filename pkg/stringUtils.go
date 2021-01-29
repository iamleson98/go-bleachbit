package pkg

import "strings"

func ctoi(c rune) byte {
	intVal := byte(c)
	return intVal
}

func isDigit(c rune) bool {
	byteVal := ctoi(c)
	return byteVal >= 48 && byteVal <= 57
}

func isLower(c rune) bool {
	byteVal := ctoi(c)
	return byteVal >= 97 && byteVal <= 122
}

func isSuper(c rune) bool {
	byteVal := ctoi(c)
	return byteVal >= 65 && byteVal < 90
}

func isAlpha(c rune) bool {
	return isSuper(c) || isLower(c)
}

func isAlNum(c rune) bool {
	return isAlpha(c) || isDigit(c)
}

func partition(s, sep string) [3]string {
	index := strings.Index(s, sep)
	if index == -1 {
		return [3]string{s, "", ""}
	}

	return [3]string{
		s[:index],
		sep,
		s[index+1:],
	}
}
