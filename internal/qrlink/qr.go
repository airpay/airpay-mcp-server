package qrlink

import (
	"bytes"
	"fmt"

	"github.com/skip2/go-qrcode"
)

// GenerateQRCodeSVG generates a QR code as SVG string.
func GenerateQRCodeSVG(content string, size int) (string, error) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("generating QR code: %w", err)
	}
	bitmap := qr.Bitmap()
	var svg bytes.Buffer
	svg.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, len(bitmap), len(bitmap), size, size))
	svg.WriteString(`<rect width="100%" height="100%" fill="#ffffff"/>`)
	for y, row := range bitmap {
		for x, module := range row {
			if module {
				svg.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1" fill="#000000"/>`, x, y))
			}
		}
	}
	svg.WriteString(`</svg>`)
	return svg.String(), nil
}
