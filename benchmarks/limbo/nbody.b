implement BenchNbody;

include "sys.m";
	sys: Sys;

include "draw.m";

include "math.m";
	math: Math;

BenchNbody: module
{
	init: fn(nil: ref Draw->Context, nil: list of string);
};

nbody(n: int): int
{
	size := 5;
	x := array[size] of {* => 0};
	y := array[size] of {* => 0};
	vx := array[size] of {* => 0};
	vy := array[size] of {* => 0};
	mass := array[size] of {* => 0};

	x[0] = 0; y[0] = 0; vx[0] = 0; vy[0] = 0; mass[0] = 1000;
	x[1] = 100; y[1] = 0; vx[1] = 0; vy[1] = 10; mass[1] = 1;
	x[2] = 200; y[2] = 0; vx[2] = 0; vy[2] = 7; mass[2] = 1;
	x[3] = 0; y[3] = 150; vx[3] = 8; vy[3] = 0; mass[3] = 1;
	x[4] = 0; y[4] = 250; vx[4] = 6; vy[4] = 0; mass[4] = 1;

	for(step := 0; step < n; step++) {
		for(i := 0; i < size; i++) {
			for(j := i+1; j < size; j++) {
				dx := x[j] - x[i];
				dy := y[j] - y[i];
				dist2 := dx*dx + dy*dy;
				if(dist2 < 1)
					dist2 = 1;
				dist := int math->sqrt(real dist2);
				if(dist < 1)
					dist = 1;
				force := mass[i] * mass[j] / dist2;
				fx := force * dx / dist;
				fy := force * dy / dist;
				vx[i] += fx / mass[i];
				vy[i] += fy / mass[i];
				vx[j] -= fx / mass[j];
				vy[j] -= fy / mass[j];
			}
		}
		for(k := 0; k < size; k++) {
			x[k] += vx[k];
			y[k] += vy[k];
		}
	}
	return x[0] + y[0] + x[1] + y[1];
}

init(nil: ref Draw->Context, nil: list of string)
{
	sys = load Sys Sys->PATH;
	math = load Math Math->PATH;

	t1 := sys->millisec();
	iterations := 20;
	result := 0;
	for(iter := 0; iter < iterations; iter++)
		result += nbody(10000);
	t2 := sys->millisec();
	sys->print("BENCH nbody %d ms %d iters %d\n", t2-t1, iterations, result);
}
