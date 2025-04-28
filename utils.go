// File: utils.go
package main

import "math"

// AutoGrid computes a grid of cols×rows to neatly hold n items
func AutoGrid(n int) (cols, rows int) {
	cols = int(math.Ceil(math.Sqrt(float64(n))))
	rows = int(math.Ceil(float64(n) / float64(cols)))
	return
}
