function [] = pcpplots(fname, initLaunch, initArrival, arrivalPlanet)
close all
% Load data file
C3 = load(sprintf('../pcpplots/contour-%s-c3.dat', fname));
vinf = load(sprintf('../pcpplots/contour-%s-vinf.dat', fname));
tof = load(sprintf('../pcpplots/contour-%s-tof.dat', fname));
dates = load(sprintf('../pcpplots/contour-%s-dates.dat', fname));
% Transpose data because it's written that way
C3 = C3';
vinf = vinf';
tof = tof';

launch_days = 0:dates(1,1):dates(1,2) - 1;
arrival_days = 0:dates(2,1):dates(2,2) - 1;

maxVinf = max(max(vinf));
if maxVinf == inf
    minVinf = round(min(min(vinf)));
    vinf_contours = [minVinf:0.5:(minVinf*2) (minVinf*2+1):(minVinf*4)];
else
    vinf_contours = round(min(min(vinf))):round((maxVinf-min(min(vinf)))/15, 1):round(maxVinf);
end
%maxC3= max(max(C3));
maxC3 = 35;
if maxC3 == inf
    minC3 = round(min(min(C3)));
    C3_contours = [minC3:3:(minC3*4) (minC3*4):10:(minC3*10)];
else
    C3_contours = round(min(min(C3))):round((maxC3-min(min(C3)))/20, 1):round(maxC3);
end

figure(1)
hold on

[cs1,h1] = contour(launch_days, arrival_days, C3, C3_contours, 'r');
clabel(cs1,h1);
[cs2,h2] = contour(launch_days, arrival_days, vinf, vinf_contours, 'b');
clabel(cs2,h2);
[cs3,h3] = contour(launch_days, arrival_days, tof, 'k');
clabel(cs3,h3);

legend('C_3 km^2/s^2', sprintf('V_{\\infty} @ %s, km/s', arrivalPlanet),'TOF, days')
xlabel(['Departure days past ' initLaunch])
ylabel(['Arrival days past ' initArrival])

end
