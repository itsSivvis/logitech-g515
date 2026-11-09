// g515keys highlights individual keys on a Logitech G515 via the HID++ 2.0
// per-key lighting feature 0x8081, keeping all other keys at their current
// (stored) color. It uses logirgb's hidpp and lighting packages.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mevdschee/logirgb/hidpp"
	"github.com/mevdschee/logirgb/lighting"
)

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "g515keys: "+format+"\n", a...)
	os.Exit(1)
}

func rgb(c int) []byte { return []byte{byte(c >> 16), byte(c >> 8), byte(c)} }

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
	keys := flag.String("keys", "wasd", "comma separated letter keys to highlight (a-z)")
	color := flag.String("color", "ff0000", "highlight color for those keys")
	base := flag.String("base", "", "color for all other keys (default: the stored effect's color)")
	flag.Parse()

	dev, info := openDevice(*devSel)
	defer dev.Close()
	fmt.Printf("device: %s [%s, %04x:%04x]\n", info.Name, info.Path, info.VendorID, info.ProductID)

	if !dev.HasFeature(hidpp.FeaturePerKeyLighting2) {
		fatal("device has no per-key lighting feature 0x8081")
	}
	c, err := lighting.New(dev)
	if err != nil {
		fatal("%v", err)
	}
	if len(c.Zones()) == 0 {
		fatal("device reports no lighting zones")
	}

	// Read the stored onboard lighting before claiming host control.
	baseColor := -1
	sp, ok, err := c.ReadProfile()
	if err != nil {
		fatal("read stored profile: %v", err)
	}
	if ok {
		fmt.Printf("stored effect: %s %v\n", sp.Effect.Name, sp.Effect.Values)
		if sp.Effect.Name != "static" {
			fmt.Printf("warning: stored effect is animated; per-key colors only show over a static base\n")
		}
		if ze, found := c.Zones()[0].Find(sp.Effect.Name); found {
			baseColor = sp.Effect.Get(ze.Type, lighting.ParamColor)
		}
	}
	if *base != "" {
		v, err := lighting.ParseColor(*base)
		if err != nil {
			fatal("base color: %v", err)
		}
		baseColor = v
	}
	if baseColor < 0 {
		baseColor = 0xFFFFFF
	}
	hiColor, err := lighting.ParseColor(*color)
	if err != nil {
		fatal("color: %v", err)
	}
	fmt.Printf("base color: #%06x, highlight: #%06x\n", baseColor, hiColor)

	markSet := map[byte]bool{}
	for _, k := range strings.Split(strings.ToLower(*keys), ",") {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		for _, ch := range k {
			if ch < 'a' || ch > 'z' {
				fatal("bad key %q (use letters a-z)", k)
			}
			markSet[byte(ch-'a'+1)] = true // zone id = letter position: a=1 .. z=26
		}
	}

	// Claim host control, then set the cluster effect to a static base color
	// (per-key colors are only visible while the cluster effect is static).
	if err := c.Claim(); err != nil {
		fatal("claim: %v", err)
	}
	st := lighting.Effect{Name: "static"}
	st.Set(lighting.ParamColor, baseColor)
	if err := c.SetEffect(0, st); err != nil {
		fatal("set static effect: %v", err)
	}

	// Zones present, from getInfo pages 0-2 (112 bit bitmap per page).
	var zones []byte
	for page := byte(0); page <= 2; page++ {
		r, err := dev.Call(hidpp.FeaturePerKeyLighting2, 0, 0x00, page)
		if err != nil {
			fatal("getInfo page %d: %v", page, err)
		}
		for i := 2; i < len(r); i++ {
			for b := 0; b < 8; b++ {
				if r[i]>>uint(b)&1 == 1 {
					z := int(page)*112 + (i-2)*8 + b
					if z >= 1 && z <= 254 {
						zones = append(zones, byte(z))
					}
				}
			}
		}
	}
	sort.Slice(zones, func(i, j int) bool { return zones[i] < zones[j] })
	fmt.Printf("%d lighting zones present\n", len(zones))

	// Paint every unmarked zone with the base color (fn6, 13 zones per call).
	var rest []byte
	for _, z := range zones {
		if !markSet[z] {
			rest = append(rest, z)
		}
	}
	for i := 0; i < len(rest); i += 13 {
		end := i + 13
		if end > len(rest) {
			end = len(rest)
		}
		params := append(rgb(baseColor), rest[i:end]...)
		if _, err := dev.Call(hidpp.FeaturePerKeyLighting2, 6, params...); err != nil {
			fatal("setRgbZonesSingleValue: %v", err)
		}
	}

	// Paint the marked keys with the highlight color (fn1, up to 4 per call).
	var marked []byte
	for z := range markSet {
		marked = append(marked, z)
	}
	sort.Slice(marked, func(i, j int) bool { return marked[i] < marked[j] })
	for i := 0; i < len(marked); i += 4 {
		end := i + 4
		if end > len(marked) {
			end = len(marked)
		}
		var params []byte
		for _, z := range marked[i:end] {
			params = append(params, z)
			params = append(params, rgb(hiColor)...)
		}
		if _, err := dev.Call(hidpp.FeaturePerKeyLighting2, 1, params...); err != nil {
			fatal("setIndividualRgbZones: %v", err)
		}
	}

	// Commit the frame (fn7 frameEnd, persistence 1 = RAM and EEPROM).
	if _, err := dev.Call(hidpp.FeaturePerKeyLighting2, 7, 0x01, 0x00, 0x00, 0x00, 0x00); err != nil {
		fatal("frameEnd: %v", err)
	}
	fmt.Printf("done: keys %q = #%06x, all other keys = #%06x\n", *keys, hiColor, baseColor)
}
