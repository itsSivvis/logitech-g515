# logitech-g515

Linux tools and a KDE Plasma 6 taskbar widget for the **Logitech G515 (LS) TKL** keyboard:

- `g515-battery` — prints the keyboard battery level in percent over HID++ 2.0 (feature `0x1004` UnifiedBattery)
- `g515keys` — per-key RGB highlighting via HID++ `0x8081`, e.g. WASD in red while all other keys keep their stored color
- `plasmoid/` — Plasma widget that shows the battery percentage in the taskbar (refreshes every 30 s, turns red at <= 20 %)

Built on [logirgb](https://github.com/mevdschee/logirgb) (MIT) by Maurits van der Schee.

Tested with the G515 LIGHTSPEED TKL (`046d:40b4` via receiver `046d:c547`, wired `046d:c355`).

## Requirements

hidraw access for the logged-in user:

```bash
echo 'KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="046d", TAG+="uaccess"' | sudo tee /etc/udev/rules.d/70-logirgb.rules
sudo udevadm control --reload-rules && sudo udevadm trigger --subsystem-match=hidraw --action=add
```

Go >= 1.26.8 to build the tools.

## Install the tools

```bash
go install github.com/itsSivvis/logitech-g515/cmd/g515-battery@latest
go install github.com/itsSivvis/logitech-g515/cmd/g515keys@latest
```

## Usage

```bash
g515-battery                                # prints e.g. "46"
g515keys -keys wasd -color ff0000           # WASD red, all other keys keep the stored color
g515keys -keys wasd -color ff0000 -base 0000ff   # explicit base color for the rest
```

## Install the Plasma widget

```bash
git clone https://github.com/itsSivvis/logitech-g515.git
kpackagetool6 --type Plasma/Applet --install logitech-g515/plasmoid
# or symlink it:
ln -s "$PWD/logitech-g515/plasmoid" ~/.local/share/plasma/plasmoids/org.chofmann.g515battery
```

Then right-click the panel → "Add Widgets…" → search for "G515".

The widget runs `g515-battery` every 30 s (looked up in `PATH`, `~/.local/bin`, `~/bin`)
and displays `G515 46%`.

## Notes

- Per-key colors are a host-side layer on top of the stored onboard profile. After a
  power cycle the keyboard falls back to its onboard lighting until `g515keys` runs again.
- HID++ 2.0 details (`0x8071` effects, `0x8081` per-key, `0x8101` profiles, `0x1004`
  battery) are documented in
  [logirgb's research notes](https://github.com/mevdschee/logirgb/tree/main/research).

## License

MIT — see [LICENSE](LICENSE). Uses logirgb (MIT).
