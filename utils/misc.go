package utils

import "fmt"

func RectToParcels(x1, y1, x2, y2 int) []string {

	minmax := func (x, y int) (int, int) {
		if x < y {
			return x, y
		}
		return y, x
	}
	x1, x2 = minmax(x1, x2)
	y1, y2 = minmax(y1, y2)

	size := (x2 - x1 + 1) * (y2 - y1 + 1)
	ret := make([]string, 0, size)
	for x := x1; x < x2 + 1; x++ {
		for y := y1; y < y2 + 1; y++ {
			ret = append(ret, fmt.Sprintf("%d,%d", x, y))
		}
	}
	return ret
}
