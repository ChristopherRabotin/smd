# Space Mission Design (smd)
Space Mission Design allows one to perform an initial space mission design, around a given celestial body or between celestial bodies.

This package was written to support my thesis and my astrodynamics courses (ASEN 6008 Space Mission Design / Interplanetary Mission Design) at the University of Colorado Boulder.

[![Build Status](https://travis-ci.org/ChristopherRabotin/smd.svg?branch=master)](https://travis-ci.org/ChristopherRabotin/smd) [![Coverage Status](https://coveralls.io/repos/ChristopherRabotin/smd/badge.svg?branch=master&service=github)](https://coveralls.io/github/ChristopherRabotin/smd?branch=master)
[![goreport](https://goreportcard.com/badge/github.com/ChristopherRabotin/smd)](https://goreportcard.com/report/github.com/ChristopherRabotin/smd)

# Features
_Note:_ this list may not be up to date with the latest developments.
- Propagation of an orbit around a celestial body
- Direct closed-loop optimization of continuous thrust via Naasz and Ruggiero control laws.
- VSOP87 support via the amazing https://github.com/soniakeys/meeus
- Patched conics for interplanetary missions
- Stream orbital elements as CSV for live visualization of how they change
- Export as a set of NASA Cosmographia files (cf. http://cosmoguide.org/) for really cool visualization of the overall mission
- Export mission state as CSV (cf. the `examples/statOD/main.go`)

# Usage
If running `smd` and planning on changing reference frames (e.g. when doing patched conics) to attempting to include third body dynamics, you will need to define the `SMD_CONFIG` environment variable. This must define whether using VSOP87 or SPICE for frame transformations. An example of such a file is found in `conf.toml`.
**Important:** this configuration file **must** be called `conf.toml` (but it can be placed in any directory).
*Note:* the availability of this file will only occur in the function which gets the heliocentric orbit of a given planet. So definitely make sure this is configured before running a long simulation or it will crash when you're looking away.
