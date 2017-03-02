import argparse
import os
from math import sqrt

import numpy as np
import spiceypy as spice

# Suppress warnings on Python 2
from sys import version as pyversion
if pyversion == '2':
    import warnings
    warnings.filterwarnings("ignore")

__kernels_loaded__ = False
def __load__kernels__():
    if __kernels_loaded__:
        return
    base_dir = os.path.dirname(os.path.abspath(__file__))
    krnls = ['de430.bsp', 'naif0012.tls', 'pck00010.tpc']
    for krnl in krnls:
        spice.furnsh(base_dir + '/spicekernels/' + krnl)

def PlanetState(planet, epoch):
    '''
    :param: planet
    :param: epoch: float or a string representing the date and time in J2000.
    :return: a numpy array
    '''
    __load__kernels__()
    if isinstance(epoch, str):
        epoch = spice.str2et(epoch)

    # Parse the planet name.
    if planet.lower() == 'mars':
        # Switch to barycenter, as per https://naif.jpl.nasa.gov/pub/naif/generic_kernels/spk/planets/aareadme_de430-de431.txt
        planet = 'Mars_barycenter'

    return spice.spkezr(planet, epoch, 'ECLIPJ2000', 'None', 'Sun')[0]

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-p', '--planet', required=True, help='planet (e.g. Earth)', type=str, dest="planet")
    parser.add_argument('-e', '--epoch', required=True, help='date time of transformation', type=str)
    args = parser.parse_args()

    print([component for component in PlanetState(args.planet, args.epoch)])
#    exp = [-996776.1190926583,-39776.102324992695,25123.28168731782,-0.5114606889356655,-0.6914491357021403,-0.34254913653144525]
#    print(nState-exp)
