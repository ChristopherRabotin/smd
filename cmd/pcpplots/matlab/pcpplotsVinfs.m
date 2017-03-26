function [] = pcpplotsVinfs(fname, initLaunch, initArrival, initPlanet, arrivalPlanet)
close all
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

minVinfI = round(min(min(vinfInit)));
maxVinfI = round(max(max(vinfInit)));
if maxVinfI > 4*minVinfI
    maxVinfI = 4*minVinfI;
end
vinfInit_contours = [minVinfI:0.25:(maxVinfI*0.3) (maxVinfI*0.3):1:maxVinfI];

minVinfA = round(min(min(vinfArrival)));
maxVinfA = round(max(max(vinfArrival)));
if maxVinfA > 4*minVinfA
    maxVinfA = 4*minVinfA;
end
vinfAr_contours = [minVinfA:0.1:(maxVinfA*0.3) (maxVinfA*0.3):1:maxVinfA];

%vinfInit_contours = [2 3 4 5 6 7 8 9 10 12 14 16 18 20 22 24];
%vinfAr_contours = [2 3 4 5 6 7 8 9 10 12 14 16 18 20 22 24];

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
