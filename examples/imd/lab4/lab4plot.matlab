% Load data file
ctrs = load('contour.dat');
x_vals = ctrs(:, 1);
y_vals = ctrs(:, 2);
C3 = ctrs(:, 3);
V_inf = ctrs(:, 4);
dt= ctrs(:, 5);

figure(1)
hold on
[cs1, h1] = contour([x_vals, y_vals, C3], 'r');
clabel(cs1 ,h1);
[cs2, h2] = contour([x_vals, y_vals, V_inf], 'b');
clabel(cs2, h2);
[cs3,h3] = contour([x_vals, y_vals, dt], 'k');
clabel(cs3,h3);