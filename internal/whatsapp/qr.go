package whatsapp

import (
	"fmt"
	"os"

	qrcode "github.com/skip2/go-qrcode"
)

const qrPNGPath = "whatsapp_qr.png"

func DisplayQR(code string) {
	err := qrcode.WriteFile(code, qrcode.Medium, 512, qrPNGPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not save QR code PNG: %v\n", err)
		return
	}
	fmt.Printf("%s\n", qrPNGPath)
}
