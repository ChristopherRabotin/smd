function [] = pcpplotsVinfs(fname, initLaunch, initArrival, initPlanet, arrivalPlanet)
% NOTE: only difference here is the name of the input files.
% Load data file
vinfInit = load(sprintf('../contour-%s-vinf-init.dat', fname));
vinfArrival = load(sprintf('../contour-%s-vinf-arrival.dat', fname));
tof = load(sprintf('../contour-%s-tof.dat', fname));
dates = load(sprintf('../contour-%s-dates.dat', fname));
% Transpose data because it's written that way
vinfInit = vinfInit';
vinfArrival = vinfArrival';
tof = tof';

launch_days = 0:dates(1,1):dates(1,2) - 1;
arrival_days = 0:dates(2,1):dates(2,2) - 1;

vinfInit_contours = round(min(min(vinfInit))):round((max(max(vinfInit))-min(min(vinfInit)))/15, 1):round(max(max(vinfInit)));
vinfAr_contours = round(min(min(vinfArrival))):round((max(max(vinfArrival))-min(min(vinfArrival)))/15, 1):round(max(max(vinfArrival)));

figure(1)
hold on

[cs1,h1] = contour(launch_days, arrival_days, vinfInit, vinfInit_contours, 'r');
clabel(cs1,h1);
[cs2,h2] = contour(launch_days, arrival_days, vinfArrival, vinfAr_contours, 'b');
clabel(cs2,h2);
[cs3,h3] = contour(launch_days, arrival_days, tof, 'k');
clabel(cs3,h3);

legend(sprintf('V_{\\infty} @ %s, km/s', initPlanet), sprintf('V_{\\infty} @ %s, km/s', arrivalPlanet),'TOF, days')
xlabel(['Departure days past ' initLaunch])
ylabel(['Arrival days past ' initArrival])

end
