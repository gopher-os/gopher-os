package hal

import (
	"gopheros/device"
	"gopheros/device/tty"
	"gopheros/device/video/console"
	"gopheros/kernel/kfmt"
)

// managedDevices contains the devices discovered by the HAL.
type managedDevices struct {
	activeConsole console.Device
	activeTTY     tty.Device
}

var devices managedDevices

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
	var drivers []device.Driver

	for _, probeFn := range hwProbeFns {
		drv := probeFn()
		if drv == nil {
			continue
		}

		strBuf.Reset()
		major, minor, patch := drv.DriverVersion()
		kfmt.Fprintf(&strBuf, "[hal] %s(%d.%d.%d): ", drv.DriverName(), major, minor, patch)
		w.prefix = strBuf.Bytes()

		if err := drv.DriverInit(&w); err != nil {
			kfmt.Fprintf(&w, "init failed: %s\n", err.Message)
			continue
		}

		kfmt.Fprintf(&w, "initialized\n")
		drivers = append(drivers, drv)
	}

	return drivers
}
