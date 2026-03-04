package main

import "os"

func main() {
	// Setenv + Getenv round-trip
	os.Setenv("GODIS_TEST", "hello42")
	val := os.Getenv("GODIS_TEST")
	println(val) // hello42

	// LookupEnv — existing key
	val2, ok := os.LookupEnv("GODIS_TEST")
	println(val2)  // hello42
	println(ok)    // true

	// LookupEnv — missing key
	_, ok2 := os.LookupEnv("GODIS_NONEXISTENT_KEY_XYZ")
	println(ok2) // false

	// Overwrite
	os.Setenv("GODIS_TEST", "updated")
	println(os.Getenv("GODIS_TEST")) // updated

	// Hostname
	host, err := os.Hostname()
	_ = err
	// On Inferno, this reads /dev/sysname
	// On native Go, this returns the real hostname
	// Just check it's non-empty
	println(len(host) > 0) // true
}
