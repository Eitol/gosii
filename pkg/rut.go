package pkg

import "strconv"

func GetRutDv(rut int) string {
	sum := 0
	multiplier := 2

	for rut > 0 {
		sum += (rut % 10) * multiplier
		rut /= 10
		multiplier++
		if multiplier == 8 {
			multiplier = 2
		}
	}

	mod := sum % 11

	if mod == 0 {
		return "0"
	} else if mod == 1 {
		return "k"
	} else {
		return strconv.Itoa(11 - mod)
	}
}
