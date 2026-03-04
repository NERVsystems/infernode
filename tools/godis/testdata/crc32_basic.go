package main

import "hash/crc32"

func main() {
	// CRC32 of empty data
	println(crc32.ChecksumIEEE([]byte{})) // 0

	// CRC32 of "hello"
	// Known value: crc32.ChecksumIEEE([]byte("hello")) = 907060870
	println(crc32.ChecksumIEEE([]byte("hello"))) // 907060870

	// CRC32 of single byte
	// crc32.ChecksumIEEE([]byte{0}) = 3523407757
	println(crc32.ChecksumIEEE([]byte{0})) // 3523407757

	// CRC32 of "123456789" — canonical test vector
	// IEEE CRC32 of "123456789" = 3421780262 (0xCBF43926)
	println(crc32.ChecksumIEEE([]byte("123456789"))) // 3421780262
}
