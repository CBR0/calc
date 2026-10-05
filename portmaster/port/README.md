## Notes

Thanks to the PortMaster crew for the WestonPack runtime, which lets apps
with a desktop toolkit run on these handhelds.

[Calc](https://github.com/CBR0/calc) is a Windows 11-style calculator
written in pure Go and drawn with MyGo's native UI (GTK3 on Linux). This
port ships the aarch64 build plus the GTK3 runtime libraries it needs
(the WestonPack runtime has cairo and pango, but no GTK), and shows it
full screen under Weston.

## Controls

| Button | Action |
|--|--|
| D-pad / left stick | Move the pointer |
| A / R1 | Click |
| B | Escape (C, clears everything) |
| X | Backspace (⌫) |
| Y / Start | Enter (=) |
| L1 | Slow pointer |
| Start + Select | Quit |
