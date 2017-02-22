format long % Yay Matlab...
% Load data file
C3 = load('../contour-c3.dat');
vinf = load('../contour-vinf.dat');
tof = load('../contour-tof.dat');
dates = load('../contour-dates.dat');
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
xlabel(['Departure days past ' dates(1, 1)])
ylabel(['Arrival days past ' dates(2, 1)])

