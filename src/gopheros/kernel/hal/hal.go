package hal

import (
	"gopheros/kernel/driver/tty"
	"gopheros/kernel/driver/video/console"
	"gopheros/kernel/hal/multiboot"
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
