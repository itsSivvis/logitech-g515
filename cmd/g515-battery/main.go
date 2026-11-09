// g515-battery prints the battery level (percent) of a Logitech G515
// keyboard over HID++ 2.0, using feature 0x1004 (UnifiedBattery) with
// 0x1000 (BatteryLevelStatus) as fallback. It uses logirgb's hidpp package.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mevdschee/logirgb/hidpp"
)

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "g515-battery: "+format+"\n", a...)
	os.Exit(1)
}

// openDevice finds the requested Logitech device and opens it, retrying a
// few times: the LIGHTSPEED receiver occasionally misses the first pings.
// Nodes matching the selector are tried first, then all other Logitech nodes
// (the receiver reaches the keyboard through a device index).
func openDevice(devSel string) (*hidpp.Device, hidpp.Info) {
	infos, err := hidpp.Discover()
	if err != nil {
		fatal("%v", err)
	}
	var matched []hidpp.Info
	for _, in := range infos {
		if strings.Contains(strings.ToLower(in.Name), strings.ToLower(devSel)) || in.Path == devSel {
			matched = append(matched, in)
		}
	}
	ordered := append(matched, infos...)
	for _, in := range ordered {
		for attempt := 0; attempt < 3; attempt++ {
			if d, err := hidpp.Open(in); err == nil {
				return d, in
			}
		}
	}
	fatal("no Logitech device matching %q responded", devSel)
	return nil, hidpp.Info{}
}

func main() {
	devSel := flag.String("d", "g515", "device: name fragment or /dev/hidraw path")
	verbose := flag.Bool("v", false, "print raw reply bytes and status")
	flag.Parse()

	dev, info := openDevice(*devSel)
	defer dev.Close()

	var level int
	switch {
	case dev.HasFeature(0x1004):
		// UnifiedBattery: fn0 getBatteryCapabilities, fn1 getBatteryInfo
		// (info reply: byte0 = percent, byte1 = level bucket, byte2 = status)
		caps, err := dev.Call(0x1004, 0x0)
		if err != nil {
			fatal("capabilities: %v", err)
		}
		r, err := dev.Call(0x1004, 0x1)
		if err != nil {
			fatal("battery info: %v", err)
		}
		if len(r) < 3 {
			fatal("short reply: % x", r)
		}
		gauge := len(caps) > 1 && caps[1]&0x02 != 0
		level = int(r[0])
		if !gauge {
			// no state-of-charge gauge: map the coarse level bucket instead
			switch r[1] {
			case 1:
				level = 5
			case 2:
				level = 25
			case 4:
				level = 75
			case 8:
				level = 100
			}
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "device %s [%s], caps: % x, info: % x, gauge=%v\n", info.Name, info.Path, caps, r, gauge)
			fmt.Fprintf(os.Stderr, "percent=%d levelBucket=%d status=%d\n", r[0], r[1], r[2])
		}
	case dev.HasFeature(0x1000):
		r, err := dev.Call(0x1000, 0x0) // BatteryLevelStatus getStatus
		if err != nil {
			fatal("%v", err)
		}
		if len(r) < 2 {
			fatal("short reply: % x", r)
		}
		level = int(r[0])
		if *verbose {
			fmt.Fprintf(os.Stderr, "device %s [%s], raw reply: % x\n", info.Name, info.Path, r)
		}
	default:
		fatal("%s has no battery feature (0x1004/0x1000)", info.Name)
	}
	if level > 100 {
		level = 100
	}
	fmt.Println(level)
}
