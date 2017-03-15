function [] = pcpplots(fname, initLaunch, initArrival)
% Load data file
C3 = load(sprintf('../contour-%s-c3.dat', fname));
vinf = load(sprintf('../contour-%s-vinf.dat', fname));
tof = load(sprintf('../contour-%s-tof.dat', fname));
dates = load(sprintf('../contour-%s-dates.dat', fname));
% Transpose data because it's written that way
C3 = C3';
vinf = vinf';
tof = tof';

launch_days = 0:dates(1,1):dates(1,2) - 1;
arrival_days = 0:dates(2,1):dates(2,2) - 1;

TOF_contours = 0:100:400;
vinf_contours = [1 2.5 3 4.5 5 7.5];
C3_contours = [1 3 5 7 10 13 16 17 19 21 25 36 55 100];

figure(1)
hold on

[cs1,h1] = contour(launch_days, arrival_days, C3, C3_contours, 'r');
clabel(cs1,h1);
[cs2,h2] = contour(launch_days, arrival_days, vinf, vinf_contours, 'b');
clabel(cs2,h2);
[cs3,h3] = contour(launch_days, arrival_days, tof, 'k');
clabel(cs3,h3);

legend('C_3 km^2/s^2','V_{\infty} @ Mars, km/s','TOF, days')
xlabel(['Departure days past ' initLaunch])
ylabel(['Arrival days past ' initArrival])

end
