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

from utils import PlanetState

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-p', '--planet', required=True, help='planet (e.g. Earth)', type=str, dest="planet")
    parser.add_argument('-e', '--epoch', required=True, help='date time of transformation', type=str)
    args = parser.parse_args()

    print([component for component in PlanetState(args.planet, args.epoch)])
