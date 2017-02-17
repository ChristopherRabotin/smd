import argparse
import os

import spiceypy as spice

__kernels_loaded__ = False

def __load__kernels__():
    if __kernels_loaded__:
        return
    base_dir = os.path.dirname(os.path.abspath(__file__))
    print(base_dir)
    krnls = ['de430.bsp', 'naif0012.tls', 'pck00010.tpc']
    for krnl in krnls:
        spice.furnsh(base_dir + '/spicekernels/' + krnl)

def DCM(fromFrame, toFrame, dt):
    __load__kernels__()
    return spice.sxform(fromFrame, toFrame, dt)

def ChgFrame(state, fromFrame, toFrame, dt):
    '''
    :param: state vector, as array of floats or numpy array.
    :param: fromFrame: spice frame name, as string
    :param: toFrame: spice frame name, as string
    :param: dt: float representing the date and time in J2000.
    :return: a numpy array
    '''
    __load__kernels__()
    if isinstance(dt, str):
        dt = spice.str2et(dt)
    stateRotd = DCM(fromFrame, toFrame, dt).dot(state)
    # Find the target body name
    if fromFrame.startswith('IAU_'):
        target = fromFrame[4:]
        obs = 'Sun'
    elif fromFrame.endswith('J2000'):
        # Works for EclipJ2000 and J2000
        target = 'Sun'
        obs = toFrame[4:]

    origin = spice.spkezr(target, dt, toFrame, 'None', obs)[0]
    return stateRotd + origin

if __name__ == '__main__':
    __load__kernels__()
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

    # Parse the epoch
    epoch = spice.str2et(args.epoch)

    new_state = ChgFrame(state, args.fromF, args.to, epoch)
    print([component for component in new_state])
