package dynamics

// Spacecraft defines a new spacecraft.
type Spacecraft struct {
	Name string // Name of spacecraft
	mass uint64 // Mass of spacecraft (in kg)
}

// GetMass returns the mass of the spacecraft at a given time.
func (s *Spacecraft) GetMass(jde float64) uint64 {
	return s.mass
}
