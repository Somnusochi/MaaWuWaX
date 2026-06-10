package combat

import "time"

// ok-ww: intro attack(1.13s) → majesty/prowess dual-state loop
//   check_majesty→perform_majesty(echo→switch) / prowess→perform_prowess(heavy) /
//   resonance triggers prowess / liberation re-checks majesty
func performAugusta(c combatActor) {
	c.attackFor(1130 * time.Millisecond)
	start := time.Now()
	timeout := 17 * time.Second
	for time.Since(start) < timeout {
		// majesty: heavy → echo → switch
		c.heavy(600 * time.Millisecond)
		// prowess: skill → heavy
		if c.skill() {
			c.heavy(600 * time.Millisecond)
		}
		if c.liberation() {
			c.heavy(600 * time.Millisecond) // re-check majesty after lib
		}
		if time.Since(start) > 14*time.Second {
			break
		}
	}
	c.echo()
	c.requestSwitch()
}
