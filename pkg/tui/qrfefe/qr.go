package qrfefe

import (
	sgr "github.com/foize/go.sgr"
	"rsc.io/qr"
)

var tops_bottoms = []rune{' ', '▀', '▄', '█'}

// A Level denotes a QR error correction level.
// From least to most tolerant of errors, they are L, M, Q, H.
type Level int

const (
	L Level = iota // 20% redundant
	M              // 38% redundant
	Q              // 55% redundant
	H              // 65% redundant
)

// Generate a text string to a QR code, which you can write to a terminal or file.
// Generate is a shorthand for DefaultConfig.Generate(text)
func Generate(size int, text string) (string, int, error) {
	// for _, c := range []Level{H, Q, M, L} {
	// 	code, err := qr.Encode(text, qr.Level(c))
	// 	if err != nil {
	// 		continue
	// 	}
	//
	// 	if code.Size > size {
	// 		continue
	// 	}
	//
	// 	return generate(c, text)
	// }

	return generate(L, text)
}

// Generate a text string to a QR code, which you can write to a terminal or file.
func generate(level Level, text string) (string, int, error) {
	code, err := qr.Encode(text, qr.Level(level))
	if err != nil {
		return "", 0, err
	}

	// rune slice
	//++ TODO: precalculate size
	qrRunes := make([]rune, 0)

	// upper border
	// addWhiteRow(&qrRunes, code.Size+4)

	// content
	for y := 0; y < code.Size-1; y += 2 {
		qrRunes = append(qrRunes, []rune(sgr.FgWhite+sgr.BgBlack)...)
		// qrRunes = append(qrRunes, '█')
		// qrRunes = append(qrRunes, '█')
		for x := 0; x < code.Size; x += 1 {
			var num int8
			if code.Black(x, y) {
				num += 1
			}

			if code.Black(x, y+1) {
				num += 2
			}
			qrRunes = append(qrRunes, tops_bottoms[num])
		}

		// qrRunes = append(qrRunes, '█')
		// qrRunes = append(qrRunes, '█')
		qrRunes = append(qrRunes, []rune(sgr.Reset)...)
		qrRunes = append(qrRunes, '\n')
	}

	// add lower border when required (only required when QR size is odd)
	addWhiteRow(&qrRunes, code.Size+4)

	return string(qrRunes), code.Size, nil
}

func addWhiteRow(qrRunes *[]rune, width int) {
	*qrRunes = append(*qrRunes, []rune(sgr.FgWhite+sgr.BgBlack)...)
	for i := 1; i < width-3; i++ {
		*qrRunes = append(*qrRunes, '▀')
	}
	*qrRunes = append(*qrRunes, []rune(sgr.Reset)...)
	*qrRunes = append(*qrRunes, '\n')
}
