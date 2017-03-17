import os
import spiceypy as spice
BARYCENTER_FRAMES = ['mars', 'jupiter', 'saturn', 'neptune', 'uranus', 'pluto']


AU = 149597870

__kernels_loaded__ = False

def _load_kernels_():
    global __kernels_loaded__
    if __kernels_loaded__:
        return
    base_dir = os.path.dirname(os.path.abspath(__file__))
    krnls = ['de430.bsp', 'naif0012.tls', 'pck00010.tpc']
    for krnl in krnls:
        spice.furnsh(base_dir + '/spicekernels/' + krnl)
    __kernels_loaded__ = True

def PlanetState(planet, epoch):
    '''
    :param: planet
    :param: epoch: float or a string representing the date and time in J2000.
    :return: a numpy array
    '''
    _load_kernels_()
    if isinstance(epoch, str):
        epoch = spice.str2et(epoch)

    # Parse the planet name.
    if planet.lower() in BARYCENTER_FRAMES:
        # Switch to barycenter, as per https://naif.jpl.nasa.gov/pub/naif/generic_kernels/spk/planets/aareadme_de430-de431.txt
        planet += '_barycenter'

    return spice.spkezr(planet, epoch, 'ECLIPJ2000', 'None', 'Sun')[0]
