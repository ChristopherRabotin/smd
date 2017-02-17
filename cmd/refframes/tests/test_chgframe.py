import unittest

import spiceypy as spice

import chgframe


class TestChgFrame(unittest.TestCase):
    '''
    This simple test ensures that the frame rotation is reversible.
    '''
    epsDecimalsR = 0
    epsDecimalsV = 6
    epochs = ["2016-03-24T20:41:48", "2016-04-14T20:50:23", "2016-05-12T18:00:15", "2018-10-02T22:21:40"]
    state = [-2.99012933e+05,-1.34647706e+05,-2.28182460e+04,-8.13323827e-02,-5.15291913e-01,-9.05957422e-02]

    def test_to_J2000(self):
        expectations = [
            [ -1.48436740e+08,  -1.06679210e+07,  -4.59223939e+06, 1.15165405e+01,  -5.01098988e+00,  -1.19598414e+01],
            [ -1.35574206e+08,  -5.84332843e+07,  -2.53537113e+07, 1.24430399e+01,  -4.95594274e-01,  -1.08624331e+01],
            [ -9.23150209e+07,  -1.09679085e+08,  -4.75325788e+07, 2.96559520e+01,   6.59711796e+00,  -7.40434589e+00],
            [  1.47374696e+08,   2.24035946e+07,   9.71621290e+06, -1.00716398e+00,   2.90284158e+00,   1.15459994e+01],
        ]
        for tno, epoch in enumerate(self.epochs):
            st = chgframe.ChgFrame(self.state, 'IAU_Earth', 'J2000', epoch)
            for i, component in enumerate(st):
                eps = self.epsDecimalsR if i < 3 else self.epsDecimalsV
                self.assertAlmostEqual(component, expectations[tno][i], eps, 'R' if i < 3 else 'V')
            # Check reversiblity
            revst = chgframe.ChgFrame(st, 'J2000', 'IAU_Earth', epoch)
            for i, component in enumerate(revst):
                eps = self.epsDecimalsR if i < 3 else self.epsDecimalsV
                self.assertAlmostEqual(component, self.state[i], eps, 'R (rev\'d)' if i < 3 else 'V (rev\'d)')

if __name__ == '__main__':
    unittest.main()
