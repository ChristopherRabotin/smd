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

def DCM(fromFrame, toFrame, dt):
    '''
    :param: fromFrame: spice frame name, as string
    :param: toFrame: spice frame name, as string
    :param: dt: float representing the date and time in J2000.
    :return: the Cartesian DCM as 3x3 np matrix
    '''
    _load_kernels_()
    return spice.pxform(fromFrame, toFrame, dt)

def ChgFrame(state, fromFrame, toFrame, dt):
    '''
    :param: state vector, as array of floats or numpy array.
    :param: fromFrame: spice frame name, as string
    :param: toFrame: spice frame name, as string
    :param: dt: float representing the date and time in J2000.
    :return: a numpy array
    '''
    _load_kernels_()
    if isinstance(dt, str):
        dt = spice.str2et(dt)

    dcm = DCM(fromFrame, toFrame, dt)
    position = dcm.dot(state[:3])
    velocity = dcm.dot(state[3:])
    stateRotd = np.array(list(position) + list(velocity))
    # Find the target body name
    if fromFrame.startswith('IAU_'):
        # From planetocentric to heliocentric
        target = fromFrame[4:]
        if target.lower() in BARYCENTER_FRAMES:
            # Switch to barycenter, as per https://naif.jpl.nasa.gov/pub/naif/generic_kernels/spk/planets/aareadme_de430-de431.txt
            target += '_barycenter'
        origin = spice.spkezr(target, dt, toFrame, 'None', 'Sun')[0]
        return stateRotd + origin

    elif fromFrame.endswith('J2000'):
        # From heliocentric to planetocentric
        # Works for EclipJ2000 and J2000
        target = toFrame[4:]
        if target.lower() in BARYCENTER_FRAMES:
            # Switch to barycenter, as per https://naif.jpl.nasa.gov/pub/naif/generic_kernels/spk/planets/aareadme_de430-de431.txt
            target += '_barycenter'
        origin = spice.spkezr(target, dt, fromFrame, 'None', 'Sun')[0]
        # Note that we perform the rotation because we asked for the origin in the *fromFrame* not in the *toFrame* like above.
        position = position - dcm.dot(origin[:3])
        velocity = velocity - dcm.dot(origin[3:])
        return np.array(list(position) + list(velocity))

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('-s', '--state', required=True, help='array of floats representing the radial and velocity vectors as [R, V]', type=str)
    parser.add_argument('-t', '--to', required=True, help='SPICE frame name the provided `state` is currently defined in', type=str)
    parser.add_argument('-f', '--from', required=True, help='SPICE frame name to transform the `state` to', type=str, dest="fromF")
    parser.add_argument('-e', '--epoch', required=True, help='date time of transformation', type=str)
    args = parser.parse_args()

    # Parse the state.
    state = []
    for component in args.state[1:-1].split(','):
        state.append(float(component.strip()))
    if len(state) != 6:
        raise ValueError("state vector must have six components")

    nState = ChgFrame(state, args.fromF, args.to, args.epoch)
    print([component for component in nState])
