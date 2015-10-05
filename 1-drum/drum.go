// Package drum is supposed to implement the decoding of .splice drum machine files.
// See golang-challenge.com/go-challenge1/ for more information
package drum

import (
	"fmt"
)

func (p *Pattern) String() string {
	s := fmt.Sprintf("Saved with HW Version: %s\nTempo: %g\n", p.Header.Version, p.Header.Tempo)

	for _, track := range p.Tracks {
		s += fmt.Sprintf("%v", track)
	}
	return s
}

func (p *track) String() string {
	s := fmt.Sprintf("(%d) %s\t", p.ID, p.Name)

	for i, note := range p.Bars {
		if i%4 == 0 {
			s += "|"
		}

		if note {
			s += "x"
		} else {
			s += "-"
		}
	}
	s += "|\n"

	return s
}
