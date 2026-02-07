implement JITBench2;

include "sys.m";
	sys: Sys;
include "draw.m";

JITBench2: module {
	init: fn(ctxt: ref Draw->Context, argv: list of string);
};

init(nil: ref Draw->Context, nil: list of string)
{
	sys = load Sys Sys->PATH;

	sys->print("=== Cross-Architecture JIT Benchmark Suite ===\n");
	sys->print("Platform: ");
	# Detect based on behavior - JIT vs interpreter timing differences
	sys->print("AMD64/ARM64 (auto-detect by timing)\n\n");

	total_start := sys->millisec();

	# Warmup
	warmup();

	# Category 1: Integer ALU operations (JIT-native on both archs)
	sys->print("== Category 1: Integer ALU ==\n");
	run_bench("1a. ADD/SUB chain", bench_addsub);
	run_bench("1b. MUL/DIV/MOD", bench_muldivmod);
	run_bench("1c. Bitwise ops", bench_bitwise);
	run_bench("1d. Shift ops", bench_shifts);
	run_bench("1e. Mixed ALU", bench_mixed_alu);

	# Category 2: Branching and control flow (JIT-native)
	sys->print("\n== Category 2: Branch & Control ==\n");
	run_bench("2a. Simple branch", bench_branch_simple);
	run_bench("2b. Compare chain", bench_compare_chain);
	run_bench("2c. Nested branches", bench_nested_branch);
	run_bench("2d. Loop countdown", bench_loop_countdown);

	# Category 3: Memory access patterns (JIT-native array ops)
	sys->print("\n== Category 3: Memory Access ==\n");
	run_bench("3a. Sequential read", bench_seq_read);
	run_bench("3b. Sequential write", bench_seq_write);
	run_bench("3c. Stride access", bench_stride_access);
	run_bench("3d. Small array hot", bench_small_array);

	# Category 4: Function calls (JIT-native CALL/RET)
	sys->print("\n== Category 4: Function Calls ==\n");
	run_bench("4a. Simple call", bench_simple_call);
	run_bench("4b. Recursive fib", bench_fib);
	run_bench("4c. Mutual recursion", bench_mutual);
	run_bench("4d. Deep call chain", bench_deep_call);

	# Category 5: Big (64-bit) operations (JIT-native on AMD64)
	sys->print("\n== Category 5: Big (64-bit) ==\n");
	run_bench("5a. Big add/sub", bench_big_addsub);
	run_bench("5b. Big bitwise", bench_big_bitwise);
	run_bench("5c. Big shifts", bench_big_shifts);
	run_bench("5d. Big comparisons", bench_big_cmp);

	# Category 6: Byte operations (JIT-native)
	sys->print("\n== Category 6: Byte Ops ==\n");
	run_bench("6a. Byte arithmetic", bench_byte_arith);
	run_bench("6b. Byte array", bench_byte_array);

	# Category 7: List operations (JIT-native pointer ops)
	sys->print("\n== Category 7: List Ops ==\n");
	run_bench("7a. List build", bench_list_build);
	run_bench("7b. List traverse", bench_list_traverse);

	# Category 8: Mixed workloads (realistic patterns)
	sys->print("\n== Category 8: Mixed Workloads ==\n");
	run_bench("8a. Sieve", bench_sieve);
	run_bench("8b. Matrix multiply", bench_matmul);
	run_bench("8c. Bubble sort", bench_bubble);
	run_bench("8d. Binary search", bench_bsearch);

	# Category 9: Type conversion overhead
	sys->print("\n== Category 9: Type Conversions ==\n");
	run_bench("9a. int<->big", bench_cvt_int_big);
	run_bench("9b. int<->byte", bench_cvt_int_byte);

	total_end := sys->millisec();
	sys->print("\n=== Total Time: %d ms ===\n", total_end - total_start);
}

ITER: con 1000000;
SMALL: con 100000;

run_bench(name: string, f: ref fn(): int)
{
	t1 := sys->millisec();
	result := f();
	t2 := sys->millisec();
	sys->print("  %-25s %6d ms  (result: %d)\n", name, t2-t1, result);
}

warmup()
{
	sum := 0;
	for (i := 0; i < 10000; i++)
		sum += i;
}

# ============================================================
# Category 1: Integer ALU
# ============================================================

bench_addsub(): int
{
	a := 0;
	for (i := 0; i < ITER; i++) {
		a += i;
		a -= (i >> 1);
		a += 3;
		a -= 1;
	}
	return a;
}

bench_muldivmod(): int
{
	a := 1;
	b := 0;
	for (i := 1; i < SMALL; i++) {
		a = (a * 3) + 1;
		b += a % 17;
		a = a / 7 + 1;
	}
	return a + b;
}

bench_bitwise(): int
{
	a := int 16rDEADBEEF;
	for (i := 0; i < ITER; i++) {
		a = a ^ (i * 16r1337);
		a = a | (a >> 16);
		a = a & int 16rFFFFFFFF;
		a = a ^ (a << 3);
	}
	return a;
}

bench_shifts(): int
{
	a := 1;
	b := 0;
	for (i := 0; i < ITER; i++) {
		a = (a << 1) | (a >> 31);  # Rotate left
		b += a & 1;
	}
	return a + b;
}

bench_mixed_alu(): int
{
	a := 0;
	b := 1;
	c := 2;
	for (i := 0; i < ITER; i++) {
		t := a + b * c;
		a = b ^ (c << 2);
		b = c - (t >> 1);
		c = t | (a & 16rFF);
	}
	return a + b + c;
}

# ============================================================
# Category 2: Branch & Control
# ============================================================

bench_branch_simple(): int
{
	count := 0;
	for (i := 0; i < ITER; i++) {
		if (i & 1)
			count++;
		else
			count--;
	}
	return count;
}

bench_compare_chain(): int
{
	count := 0;
	for (i := 0; i < ITER; i++) {
		v := i % 100;
		if (v < 25)
			count += 1;
		else if (v < 50)
			count += 2;
		else if (v < 75)
			count += 3;
		else
			count += 4;
	}
	return count;
}

bench_nested_branch(): int
{
	count := 0;
	for (i := 0; i < SMALL; i++) {
		a := i % 10;
		b := i % 7;
		if (a > 5) {
			if (b > 3)
				count += a + b;
			else
				count += a - b;
		} else {
			if (b > 3)
				count -= a + b;
			else
				count -= a - b;
		}
	}
	return count;
}

bench_loop_countdown(): int
{
	sum := 0;
	for (outer := 0; outer < 1000; outer++) {
		n := 1000;
		while (n > 0) {
			sum += n;
			n--;
		}
	}
	return sum;
}

# ============================================================
# Category 3: Memory Access
# ============================================================

bench_seq_read(): int
{
	arr := array[1000] of int;
	for (i := 0; i < 1000; i++)
		arr[i] = i;

	sum := 0;
	for (iter := 0; iter < 1000; iter++)
		for (i = 0; i < 1000; i++)
			sum += arr[i];
	return sum;
}

bench_seq_write(): int
{
	arr := array[1000] of int;
	for (iter := 0; iter < 1000; iter++)
		for (i := 0; i < 1000; i++)
			arr[i] = i + iter;
	return arr[999];
}

bench_stride_access(): int
{
	arr := array[1024] of int;
	for (i := 0; i < 1024; i++)
		arr[i] = i;

	sum := 0;
	for (iter := 0; iter < 1000; iter++) {
		for (stride := 1; stride <= 8; stride *= 2)
			for (i = 0; i < 1024; i += stride)
				sum += arr[i];
	}
	return sum;
}

bench_small_array(): int
{
	arr := array[16] of int;
	for (i := 0; i < 16; i++)
		arr[i] = i;

	sum := 0;
	for (iter := 0; iter < SMALL; iter++)
		for (i = 0; i < 16; i++)
			sum += arr[i];
	return sum;
}

# ============================================================
# Category 4: Function Calls
# ============================================================

bench_simple_call(): int
{
	sum := 0;
	for (i := 0; i < ITER; i++)
		sum += add_one(i);
	return sum;
}

add_one(x: int): int
{
	return x + 1;
}

bench_fib(): int
{
	sum := 0;
	for (i := 0; i < 50; i++)
		sum += fib(25);
	return sum;
}

fib(n: int): int
{
	if (n <= 1) return n;
	return fib(n-1) + fib(n-2);
}

bench_mutual(): int
{
	sum := 0;
	for (i := 0; i < SMALL; i++)
		sum += is_even(i % 20);
	return sum;
}

is_even(n: int): int
{
	if (n == 0) return 1;
	return is_odd(n - 1);
}

is_odd(n: int): int
{
	if (n == 0) return 0;
	return is_even(n - 1);
}

bench_deep_call(): int
{
	sum := 0;
	for (i := 0; i < SMALL; i++)
		sum += chain_a(i, 10);
	return sum;
}

chain_a(x, depth: int): int
{
	if (depth <= 0) return x;
	return chain_b(x + 1, depth - 1);
}

chain_b(x, depth: int): int
{
	if (depth <= 0) return x;
	return chain_a(x + 1, depth - 1);
}

# ============================================================
# Category 5: Big (64-bit)
# ============================================================

bench_big_addsub(): int
{
	a := big 0;
	b := big 1;
	for (i := 0; i < ITER; i++) {
		a += b;
		b = a - b;
		a += big 3;
	}
	return int (a & big 16rFFFFFFFF);
}

bench_big_bitwise(): int
{
	a := big 16rDEADBEEFCAFEBABE;
	for (i := 0; i < ITER; i++) {
		a = a ^ (big i * big 16r1337);
		a = a | (a >> 16);
		a = a & big 16rFFFFFFFFFFFFFFFF;
	}
	return int (a & big 16rFFFFFFFF);
}

bench_big_shifts(): int
{
	a := big 1;
	count := 0;
	for (i := 0; i < ITER; i++) {
		a = (a << 1) | (a >> 63);
		if (int (a & big 1))
			count++;
	}
	return count;
}

bench_big_cmp(): int
{
	count := 0;
	a := big 0;
	b := big 1;
	for (i := 0; i < ITER; i++) {
		if (a < b) count++;
		if (a == big 0) count++;
		if (b != big 0) count++;
		a += big 1;
		b += big 2;
	}
	return count;
}

# ============================================================
# Category 6: Byte Ops
# ============================================================

bench_byte_arith(): int
{
	a := byte 0;
	sum := 0;
	for (i := 0; i < ITER; i++) {
		a = a + byte 7;
		a = a ^ byte 16r55;
		a = a & byte 16rFE;
		sum += int a;
	}
	return sum;
}

bench_byte_array(): int
{
	arr := array[256] of byte;
	for (i := 0; i < 256; i++)
		arr[i] = byte i;

	sum := 0;
	for (iter := 0; iter < 10000; iter++)
		for (i = 0; i < 256; i++)
			sum += int arr[i];
	return sum;
}

# ============================================================
# Category 7: List Ops
# ============================================================

bench_list_build(): int
{
	count := 0;
	for (iter := 0; iter < 1000; iter++) {
		l : list of int = nil;
		for (i := 0; i < 100; i++)
			l = i :: l;
		count += len l;
	}
	return count;
}

bench_list_traverse(): int
{
	# Build a list once
	l : list of int = nil;
	for (i := 0; i < 1000; i++)
		l = i :: l;

	sum := 0;
	for (iter := 0; iter < 1000; iter++) {
		for (tmp := l; tmp != nil; tmp = tl tmp)
			sum += hd tmp;
	}
	return sum;
}

# ============================================================
# Category 8: Mixed Workloads
# ============================================================

bench_sieve(): int
{
	SIZE: con 10000;
	sieve := array[SIZE] of int;

	count := 0;
	for (iter := 0; iter < 100; iter++) {
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

bench_matmul(): int
{
	N: con 32;
	a := array[N*N] of int;
	b := array[N*N] of int;
	c := array[N*N] of int;

	for (i := 0; i < N*N; i++) {
		a[i] = i % 7;
		b[i] = i % 11;
	}

	for (iter := 0; iter < 100; iter++) {
		for (i = 0; i < N; i++)
			for (j := 0; j < N; j++) {
				s := 0;
				for (k := 0; k < N; k++)
					s += a[i*N + k] * b[k*N + j];
				c[i*N + j] = s;
			}
	}
	return c[0] + c[N*N-1];
}

bench_bubble(): int
{
	N: con 500;
	arr := array[N] of int;

	for (iter := 0; iter < 10; iter++) {
		# Fill in reverse
		for (i := 0; i < N; i++)
			arr[i] = N - i;

		# Bubble sort
		for (i = 0; i < N - 1; i++)
			for (j := 0; j < N - 1 - i; j++)
				if (arr[j] > arr[j+1]) {
					t := arr[j];
					arr[j] = arr[j+1];
					arr[j+1] = t;
				}
	}
	return arr[0] + arr[N-1];
}

bench_bsearch(): int
{
	N: con 10000;
	arr := array[N] of int;
	for (i := 0; i < N; i++)
		arr[i] = i * 3;

	found := 0;
	for (iter := 0; iter < SMALL; iter++) {
		target := (iter * 7) % (N * 3);
		lo := 0;
		hi := N - 1;
		while (lo <= hi) {
			mid := (lo + hi) / 2;
			if (arr[mid] == target) {
				found++;
				break;
			} else if (arr[mid] < target)
				lo = mid + 1;
			else
				hi = mid - 1;
		}
	}
	return found;
}

# ============================================================
# Category 9: Type Conversions
# ============================================================

bench_cvt_int_big(): int
{
	sum := 0;
	for (i := 0; i < ITER; i++) {
		b := big i;
		b = b + big 42;
		sum += int b;
	}
	return sum;
}

bench_cvt_int_byte(): int
{
	sum := 0;
	for (i := 0; i < ITER; i++) {
		b := byte i;
		b = b + byte 1;
		sum += int b;
	}
	return sum;
}
