import argparse
import os
from math import sqrt

import numpy as np
import spiceypy as spice

from dateutil.parser import parse as parsedt
from datetime import timedelta

# Suppress warnings on Python 2
from sys import version as pyversion
if pyversion == '2':
    import warnings
    warnings.filterwarnings("ignore")

from utils import PlanetState, _load_kernels_

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-p', '--planet', required=True, help='planet (e.g. Earth)', type=str, dest="planet")
    parser.add_argument('-s', '--start', required=True, help='the start date time of transformation', type=str)
    parser.add_argument('-e', '--end', required=True, help='the end date time of transformation', type=str)
    parser.add_argument('-r', '--reso', required=True, help='the resolution transformation', type=str)
    args = parser.parse_args()

    # Parse the resolution
    unit = args.reso[-1]
    resolution_units = {'d': 'days', 'm': 'minutes', 's': 'seconds'}
    if unit not in resolution_units:
        raise ValueError("unknown unit " + unit)
    reso_num = int(args.reso[0]) # Let it raise.
    deltaargs = {resolution_units[unit]: reso_num}
    start_date = parsedt(args.start)
    end_date = parsedt(args.end)
    if start_date.year != end_date.year:
        raise ValueError("must generate year by year")
    end_date += timedelta(days=1)

    _load_kernels_()
    f = open('../../data/horizon/' + args.planet + '-' + str(start_date.year) + '.csv', 'w')
    prev_month = 0
    while start_date <= end_date:
        if prev_month != start_date.month:
            prev_month = start_date.month
            print('Generating for month ' + str(prev_month))
        date_str = '{0.year}-{0.month}-{0.day}T{0.hour}:{0.minute}:{0.second}.{0.microsecond}'.format(start_date)
        et = spice.str2et(date_str)
        jde = spice.j2000() + et/spice.spd()
        f.write(str(jde) + ','+ date_str + ',')
        f.write(','.join([str(component) for component in PlanetState(args.planet, et)]))
        f.write('\n')
        start_date = start_date + timedelta(**deltaargs)
