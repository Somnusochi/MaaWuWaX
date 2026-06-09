// Package keycode provides macOS CGKeyCode mappings for Wuthering Waves keybinds.
package keycode

import "fmt"

// CGKeyCode values used by MaaMacOSControlUnit PostToPid / GlobalEvent.
// Reference: /System/Library/Frameworks/Carbon.framework/Versions/A/Frameworks/HIToolbox.framework/Versions/A/Headers/Events.h
var KeyCodes = map[string]int32{
	"A":     0,
	"S":     1,
	"D":     2,
	"F":     3,
	"H":     4,
	"G":     5,
	"Z":     6,
	"X":     7,
	"C":     8,
	"V":     9,
	"B":     11,
	"Q":     12,
	"W":     13,
	"E":     14,
	"R":     15,
	"Y":     16,
	"T":     17,
	"1":     18,
	"2":     19,
	"3":     20,
	"4":     21,
	"6":     22,
	"5":     23,
	"=":     24,
	"9":     25,
	"7":     26,
	"-":     27,
	"8":     28,
	"0":     29,
	"O":     31,
	"U":     32,
	"[":     33,
	"I":     34,
	"P":     35,
	"L":     37,
	"J":     38,
	"K":     40,
	"N":     45,
	"M":     46,
	"TAB":   48,
	"SPACE": 49,
	"ESC":   53,
	"SHIFT": 56,
	"CAPS":  57,
	"CTRL":  59,
	"CMD":   55,
	"ALT":   58,
	"RETURN": 36,
	"DELETE": 51,
	"F1":    122,
	"F2":    120,
	"F3":    99,
	"F4":    118,
	"F5":    96,
	"F6":    97,
	"F7":    98,
	"F8":    100,
	"F9":    101,
	"F10":   109,
	"F11":   103,
	"F12":   111,
}

// Code returns the CGKeyCode for the given key name.
// Returns an error if the key is not found.
func Code(name string) (int32, error) {
	if code, ok := KeyCodes[name]; ok {
		return code, nil
	}
	return 0, fmt.Errorf("keycode: unknown key %q", name)
}

// MustCode returns the CGKeyCode for the given key name.
// Panics if the key is not found.
func MustCode(name string) int32 {
	code, err := Code(name)
	if err != nil {
		panic(err)
	}
	return code
}
