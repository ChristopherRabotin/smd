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

from utils import BARYCENTER_FRAMES, _load_kernels_

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

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-p', '--planet', required=True, help='planet (e.g. Earth)', type=str, dest="planet")
    parser.add_argument('-e', '--epoch', required=True, help='date time of transformation', type=str)
    args = parser.parse_args()

    print([component for component in PlanetState(args.planet, args.epoch)])
