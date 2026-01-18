package whatsapp

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	qrcode "github.com/skip2/go-qrcode"
)

const qrPNGPath = "whatsapp_qr.png"

func DisplayQR(code string) {
	// Generate and save QR code as PNG
	err := qrcode.WriteFile(code, qrcode.Medium, 512, qrPNGPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not save QR code PNG: %v\n", err)
		return
	}

	fmt.Printf("Opening QR code in default image viewer...\n")

	// Open the PNG file with the system's default image viewer
	if err := openFile(qrPNGPath); err != nil {
		fmt.Fprintf(os.Stderr, "Could not auto-open QR code. Please manually open: %s\n", qrPNGPath)
	}
}

// openFile opens a file with the system's default application
func openFile(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}
