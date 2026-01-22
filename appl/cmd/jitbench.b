implement JITBench;

include "sys.m";
	sys: Sys;
include "draw.m";

JITBench: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

ITERATIONS: con 10000000;
SMALL_ITER: con 1000000;

init(nil: ref Draw->Context, nil: list of string)
{
	sys = load Sys Sys->PATH;

	sys->print("=== JIT Benchmark Suite ===\n");
	sys->print("Iterations: %d (arithmetic), %d (other)\n\n", ITERATIONS, SMALL_ITER);

	# Warm up
	warmup();

	# Run benchmarks
	t0 := sys->millisec();

	sys->print("1. Integer Arithmetic\n");
	t1 := sys->millisec();
	arith_result := bench_arithmetic();
	t2 := sys->millisec();
	sys->print("   Result: %d, Time: %d ms\n\n", arith_result, t2-t1);

	sys->print("2. Loop with Array Access\n");
	t1 = sys->millisec();
	array_result := bench_array();
	t2 = sys->millisec();
	sys->print("   Result: %d, Time: %d ms\n\n", array_result, t2-t1);

	sys->print("3. Function Calls\n");
	t1 = sys->millisec();
	call_result := bench_calls();
	t2 = sys->millisec();
	sys->print("   Result: %d, Time: %d ms\n\n", call_result, t2-t1);

	sys->print("4. Fibonacci (recursive)\n");
	t1 = sys->millisec();
	fib_result := bench_fib();
	t2 = sys->millisec();
	sys->print("   Result: %d, Time: %d ms\n\n", fib_result, t2-t1);

	sys->print("5. Sieve of Eratosthenes\n");
	t1 = sys->millisec();
	sieve_result := bench_sieve();
	t2 = sys->millisec();
	sys->print("   Result: %d primes, Time: %d ms\n\n", sieve_result, t2-t1);

	sys->print("6. Nested Loops\n");
	t1 = sys->millisec();
	nested_result := bench_nested();
	t2 = sys->millisec();
	sys->print("   Result: %d, Time: %d ms\n\n", nested_result, t2-t1);

	tend := sys->millisec();
	sys->print("=== Total Time: %d ms ===\n", tend-t0);
}

warmup()
{
	# Warm up the JIT
	sum := 0;
	for (i := 0; i < 10000; i++)
		sum += i;
}

bench_arithmetic(): int
{
	a := 1;
	b := 2;
	c := 3;

	for (i := 0; i < ITERATIONS; i++) {
		a = a + b;
		b = b * 3;
		c = c - a;
		a = a ^ b;
		b = b & 16rFFFF;
		c = c | 16r1;
		a = a << 1;
		b = b >> 1;
		c = c + (a % 17);
	}
	return a + b + c;
}

bench_array(): int
{
	arr := array[1000] of int;

	# Initialize
	for (i := 0; i < 1000; i++)
		arr[i] = i;

	sum := 0;
	for (j := 0; j < SMALL_ITER; j++) {
		for (i := 0; i < 1000; i++)
			sum += arr[i];
	}
	return sum;
}

bench_calls(): int
{
	sum := 0;
	for (i := 0; i < SMALL_ITER; i++)
		sum += helper_add(i, i+1);
	return sum;
}

helper_add(a, b: int): int
{
	return a + b;
}

bench_fib(): int
{
	sum := 0;
	for (i := 0; i < 100; i++)
		sum += fib(25);
	return sum;
}

fib(n: int): int
{
	if (n <= 1)
		return n;
	return fib(n-1) + fib(n-2);
}

bench_sieve(): int
{
	SIZE: con 100000;
	sieve := array[SIZE] of int;

	count := 0;
	for (iter := 0; iter < 10; iter++) {
		# Initialize
		for (i := 0; i < SIZE; i++)
			sieve[i] = 1;

		sieve[0] = 0;
		sieve[1] = 0;

		for (i = 2; i * i < SIZE; i++) {
			if (sieve[i]) {
				for (j := i * i; j < SIZE; j += i)
					sieve[j] = 0;
			}
		}

		count = 0;
		for (i = 0; i < SIZE; i++)
			if (sieve[i])
				count++;
	}
	return count;
}

bench_nested(): int
{
	sum := 0;
	for (i := 0; i < 500; i++)
		for (j := 0; j < 500; j++)
			for (k := 0; k < 200; k++)
				sum += i + j + k;
	return sum;
}
