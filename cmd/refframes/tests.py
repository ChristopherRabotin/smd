import unittest

import spiceypy as spice

from chgframe import ChgFrame
from heliostate import PlanetState


class TestChgFrame(unittest.TestCase):
    '''
    This simple test ensures that the frame rotation is reversible.
    '''
    epsDecimalsR = 0
    epsDecimalsV = 6
    epochs = ["2016-03-24T20:41:48", "2016-04-14T20:50:23", "2016-05-12T18:00:15", "2018-10-02T22:21:40"]
    state = [-996776.1190926583,-39776.102324992695,25123.28168731782,-0.5114606889356655,-0.6914491357021403,-0.34254913653144525]

    def test_to_EclipJ2000(self):
        expectations = [
            [-148030923.95108017, -12123548.951590259, 302492.17670564854, 2.6590243298160754, -29.849194304414752, -0.35374685933315347],
            [-134976740.1465185, -64019984.225526303, 173268.29557571266, 12.914400364190689, -26.830454218461472, -0.48247193678732703],
            [-91834879.055177972, -119989870.393112, 265756.61887669913, 23.896559117820097, -18.302614365094886, -0.39983700293859631],
            [146717590.82034796, 24671383.129385762, -52708.613323677433, -6.003251556028836, 28.634454862602379, -0.09470460569670297],
        ]
        for tno, epoch in enumerate(self.epochs):
            st = ChgFrame(self.state, 'IAU_Earth', 'EclipJ2000', epoch)
            for i, component in enumerate(st):
                eps = self.epsDecimalsR if i < 3 else self.epsDecimalsV
                self.assertAlmostEqual(component, expectations[tno][i], eps, '{} {}[{}]'.format(epoch, 'R' if i < 3 else 'V', i))
            # Check reversiblity
            revst = ChgFrame(st, 'EclipJ2000', 'IAU_Earth', epoch)
            for i, component in enumerate(revst):
                eps = self.epsDecimalsR if i < 3 else self.epsDecimalsV
                self.assertAlmostEqual(component, self.state[i], eps, '{} {}'.format(epoch, 'R (rev\'d)' if i < 3 else 'V (rev\'d)'))

class TestPlanetState(unittest.TestCase):
    def test_Mars(self):
        # We test Mars specifically because the kernel uses a different name for this.
        exp = [17278277.787544131, 232608811.57426503, 4449867.5818615705, -23.242444259084987, 3.8523561650721909, 0.65118719284938198]
        got = PlanetState("mArS", "2015-06-20 00:00:00")
        self.assertTrue(all([val == exp[i] for i, val in enumerate(got)]), 'incorrect Mars state returned')

if __name__ == '__main__':
    unittest.main()
