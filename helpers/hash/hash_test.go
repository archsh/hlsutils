package hash

import (
	"testing"
	"fmt"
)


func TestCRC32(t *testing.T) {
	fmt.Println("CRC32:", CRC32("This is a test!"))
}

func TestSDBM(t *testing.T) {
	fmt.Println("CRC32:", SDBM("This is a test!"))
}

func TestWT6(t *testing.T) {
	fmt.Println("CRC32:", WT6("This is a test!"))
}

func TestDJB2(t *testing.T) {
	fmt.Println("CRC32:", DJB2("This is a test!"))
}

func BenchmarkCRC32(b *testing.B) {

}

func BenchmarkSDBM(b *testing.B) {

}

func BenchmarkWT6(b *testing.B) {

}

func BenchmarkDJB2(b *testing.B) {

}