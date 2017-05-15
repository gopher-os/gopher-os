package hal

import (
	"github.com/achilleasa/gopher-os/kernel/driver/tty"
	"github.com/achilleasa/gopher-os/kernel/driver/video/console"
	"github.com/achilleasa/gopher-os/kernel/hal/multiboot"
)

var (
	egaConsole = &console.Ega{}

	// ActiveTerminal points to the currently active terminal.
	ActiveTerminal = &tty.Vt{}
)

// InitTerminal provides a basic terminal to allow the kernel to emit some output
// till everything is properly setup
func InitTerminal() {
	fbInfo := multiboot.GetFramebufferInfo()

	egaConsole.Init(uint16(fbInfo.Width), uint16(fbInfo.Height), uintptr(fbInfo.PhysAddr))
	ActiveTerminal.AttachTo(egaConsole)
}
