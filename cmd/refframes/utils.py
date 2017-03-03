import os
import spiceypy as spice
BARYCENTER_FRAMES = ['mars', 'jupiter', 'saturn', 'neptune', 'uranus', 'pluto']


AU = 149597870

__kernels_loaded__ = False
def _load_kernels_():
    if __kernels_loaded__:
        return
    base_dir = os.path.dirname(os.path.abspath(__file__))
    krnls = ['de430.bsp', 'naif0012.tls', 'pck00010.tpc']
    for krnl in krnls:
        spice.furnsh(base_dir + '/spicekernels/' + krnl)
