package hal

import (
	"bytes"
	"gopheros/device"
	"gopheros/device/tty"
	"gopheros/device/video/console"
	"gopheros/device/video/console/font"
	"gopheros/device/video/console/logo"
	"gopheros/kernel/hal/multiboot"
	"gopheros/kernel/kfmt"
	"sort"
)

// managedDevices contains the devices discovered by the HAL.
type managedDevices struct {
	activeConsole console.Device
	activeTTY     tty.Device

	// activeDrivers tracks all initialized device drivers.
	activeDrivers []device.Driver
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
	// Get driver list and sort by detection priority
	drivers := device.DriverList()
	sort.Sort(drivers)

	probe(drivers)
}

// probe executes the probe function for each driver and invokes
// onDriverInit for each successfully initialized driver.
func probe(driverInfoList device.DriverInfoList) {
	var w = kfmt.PrefixWriter{Sink: kfmt.GetOutputSink()}

	for _, info := range driverInfoList {
		drv := info.Probe()
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
		onDriverInit(info, drv)
		devices.activeDrivers = append(devices.activeDrivers, drv)
	}
}

// onDriverInit is invoked by probe() whenever a piece of hardware is detected
// and successfully initialized.
func onDriverInit(info *device.DriverInfo, drv device.Driver) {
	switch drvImpl := drv.(type) {
	case console.Device:
		onConsoleInit(drvImpl)
	case tty.Device:
		if devices.activeTTY != nil {
			return
		}

		devices.activeTTY = drvImpl
		if devices.activeConsole != nil {
			linkTTYToConsole()
		}
	}
}

// onConsoleInit is invoked whenever a console is initialized. If this is the
// first found console it automatically becomes the active console. In
// addition, if the console supports fonts and/or logos this function ensures
// that they are loaded and attached to the console. Finally, if an active TTY
// device is present, it will be automatically linked to the first active
// console via a call to linkTTYToConsole.
func onConsoleInit(cons console.Device) {
	if devices.activeConsole != nil {
		return
	}

	devices.activeConsole = cons

	if logoSetter, ok := (devices.activeConsole).(console.LogoSetter); ok {
		disableLogo := false
		for k, v := range multiboot.GetBootCmdLine() {
			if k == "consoleLogo" && v == "off" {
				disableLogo = true
				break
			}
		}

		if !disableLogo {
			consW, consH := devices.activeConsole.Dimensions(console.Pixels)
			logoSetter.SetLogo(logo.BestFit(consW, consH))
		}
	}

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

	if devices.activeTTY != nil {
		linkTTYToConsole()
	}
}

// linkTTYToConsole connects the active TTY device to the active console device
// and syncs their contents.
func linkTTYToConsole() {
	devices.activeTTY.AttachTo(devices.activeConsole)
	kfmt.SetOutputSink(devices.activeTTY)

	// Sync terminal contents with console
	devices.activeTTY.SetState(tty.StateActive)

}
