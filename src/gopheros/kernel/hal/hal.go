package hal

import (
	"bytes"
	"gopheros/device"
	"gopheros/device/tty"
	"gopheros/device/video/console"
	"gopheros/device/video/console/font"
	"gopheros/kernel/hal/multiboot"
	"gopheros/kernel/kfmt"
)

// managedDevices contains the devices discovered by the HAL.
type managedDevices struct {
	activeConsole console.Device
	activeTTY     tty.Device
}

var (
	devices managedDevices
	strBuf  bytes.Buffer
)

// ActiveTTY returns the currently active TTY
func ActiveTTY() tty.Device {
	return devices.activeTTY
}

// DetectHardware probes for hardware devices and initializes the appropriate
// drivers.
func DetectHardware() {
	consoles := probe(console.ProbeFuncs)
	if len(consoles) != 0 {
		devices.activeConsole = consoles[0].(console.Device)

		if fontSetter, ok := (devices.activeConsole).(console.FontSetter); ok {
			consW, consH := devices.activeConsole.Dimensions(console.Pixels)

			// Check boot cmdline for a font request
			var selFont *font.Font
			for k, v := range multiboot.GetBootCmdLine() {
				if k != "consoleFont" {
					continue
				}

				if selFont = font.FindByName(v); selFont != nil {
					break
				}
			}

			if selFont == nil {
				selFont = font.BestFit(consW, consH)
			}

			fontSetter.SetFont(selFont)
		}
	}

	ttys := probe(tty.ProbeFuncs)
	if len(ttys) != 0 {
		devices.activeTTY = ttys[0].(tty.Device)
		devices.activeTTY.AttachTo(devices.activeConsole)
		kfmt.SetOutputSink(devices.activeTTY)

		// Sync terminal contents with console
		devices.activeTTY.SetState(tty.StateActive)
	}
}

// probe executes the supplied hw probe functions and attempts to initialize
// each detected device. The function returns a list of device drivers that
// were successfully initialized.
func probe(hwProbeFns []device.ProbeFn) []device.Driver {
	var (
		drivers []device.Driver
		w       = kfmt.PrefixWriter{Sink: kfmt.GetOutputSink()}
	)

	for _, probeFn := range hwProbeFns {
		drv := probeFn()
		if drv == nil {
			continue
		}

		strBuf.Reset()
		major, minor, patch := drv.DriverVersion()
		kfmt.Fprintf(&strBuf, "[hal] %s(%d.%d.%d): ", drv.DriverName(), major, minor, patch)
		w.Prefix = strBuf.Bytes()

		if err := drv.DriverInit(&w); err != nil {
			kfmt.Fprintf(&w, "init failed: %s\n", err.Message)
			continue
		}

		kfmt.Fprintf(&w, "initialized\n")
		drivers = append(drivers, drv)
	}

	return drivers
}
