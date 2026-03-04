package main

import "encoding/json"

func main() {
	// Valid JSON
	println(json.Valid([]byte(`{"key":"value"}`)))
	println(json.Valid([]byte(`[1,2,3]`)))
	println(json.Valid([]byte(`"hello"`)))
	println(json.Valid([]byte(`42`)))
	println(json.Valid([]byte(`true`)))
	println(json.Valid([]byte(`null`)))
	println(json.Valid([]byte(`{"a":{"b":[1,2]}}`)))

	// Invalid JSON
	println(json.Valid([]byte(``)))
	println(json.Valid([]byte(`{`)))
	println(json.Valid([]byte(`}`)))
	println(json.Valid([]byte(`{"key":"unterminated`)))
	println(json.Valid([]byte(`[1,2,3}`)))
}
