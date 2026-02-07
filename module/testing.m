Testing: module
{
	PATH: con "/dis/lib/testing.dis";

	# Test context passed to each test function
	T: adt
	{
		name:    string;
		srcfile: string;		# source file for clickable addresses (optional)
		failed:  int;
		skipped: int;
		output:  list of string;	# accumulated log output
		start:   int;			# start time in milliseconds

		# Logging
		log:     fn(t: self ref T, msg: string);

		# Failure reporting (continues test execution)
		error:   fn(t: self ref T, msg: string);

		# Fatal failure (stops test via exception)
		fatal:   fn(t: self ref T, msg: string);

		# Skip test (stops test via exception)
		skip:    fn(t: self ref T, msg: string);

		# Assertions - return 1 on success, 0 on failure (and mark test failed)
		assert:     fn(t: self ref T, cond: int, msg: string): int;
		asserteq:   fn(t: self ref T, got, want: int, msg: string): int;
		assertne:   fn(t: self ref T, got, notexpect: int, msg: string): int;
		assertseq:  fn(t: self ref T, got, want: string, msg: string): int;
		assertsne:  fn(t: self ref T, got, notexpect: string, msg: string): int;
		assertnil:  fn(t: self ref T, got: string, msg: string): int;
		assertnotnil: fn(t: self ref T, got: string, msg: string): int;
	};

	# Initialize the testing framework
	init:     fn();

	# Create a new T for a named test
	# srcfile is optional - if provided, enables clickable file:/pattern/ addresses
	newT:     fn(name: string): ref T;
	newTsrc:  fn(name, srcfile: string): ref T;

	# Run a test function and handle exceptions
	# Call this wrapper around each test function:
	#   testing->run(t, testMyFunc, (t,));
	# Returns 1 on pass, 0 on fail/skip
	done:     fn(t: ref T): int;

	# Print final summary - call after all tests complete
	# Pass counts of (passed, failed, skipped)
	# Returns failed count (for use as exit status)
	summary:  fn(passed, failed, skipped: int): int;

	# Configuration
	verbose:  fn(v: int);		# set verbose mode (default 0)
	getverbose: fn(): int;		# get verbose setting
};
