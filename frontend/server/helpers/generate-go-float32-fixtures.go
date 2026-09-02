// Generates go-float32-fixtures.json. Run from frontend/server with:
// go run ./helpers/generate-go-float32-fixtures.go
package main

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
)

type fixture struct {
	Bits     string `json:"bits"`
	Expected string `json:"expected"`
}

func main() {
	bits := []uint32{
		0x00000000, 0x80000000, 0x00000001, 0x007fffff, 0x00800000,
		0x3f7fffff, 0x3f800000, 0x3f800001, 0x40000000, 0x4b7fffff,
		0x4b800000, 0x4b800001, 0x497423f0, 0x3eaaaaab, 0x3727c5ac,
		0x7f7fffff, 0xff7fffff, 0x7f800000, 0xff800000, 0x7fc00000,
	}
	state := uint32(0x9e3779b9)
	for i := 0; i < 300; i++ {
		state = state*1664525 + 1013904223
		bits = append(bits, state)
	}
	fixtures := make([]fixture, 0, len(bits))
	for _, b := range bits {
		f := math.Float32frombits(b)
		fixtures = append(fixtures, fixture{
			Bits:     string([]byte{hexDigit(b >> 28), hexDigit(b >> 24), hexDigit(b >> 20), hexDigit(b >> 16), hexDigit(b >> 12), hexDigit(b >> 8), hexDigit(b >> 4), hexDigit(b)}),
			Expected: strconv.FormatFloat(float64(f), 'g', -1, 32),
		})
	}
	data, err := json.MarshalIndent(fixtures, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("helpers/go-float32-fixtures.json", append(data, '\n'), 0644); err != nil {
		panic(err)
	}
}

func hexDigit(v uint32) byte {
	v &= 15
	if v < 10 {
		return byte('0' + v)
	}
	return byte('a' + v - 10)
}
