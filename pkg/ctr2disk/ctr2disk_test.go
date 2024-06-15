package ctr2disk

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_dd(t *testing.T) {
	testCases := []struct {
		of    string
		bs    int
		count int
	}{
		{
			of:    "test-dd-1",
			bs:    4194304,
			count: 512,
		},
	}
	for _, tc := range testCases {
		// Ensure first in case it exists.
		err := os.Remove(tc.of)
		if !(err == nil || os.IsNotExist(err)) {
			t.Fatal(err)
		}

		err = dd(tc.of, tc.bs, tc.count)
		assert.Nil(t, err)

		err = os.Remove(tc.of)
		if !(err == nil || os.IsNotExist(err)) {
			t.Fatal(err)
		}
	}
}

func Benchmark_dd_1(b *testing.B) {
	benchmark_dd("test-dd-1", 8388608, 256, b)
}

func Benchmark_dd_2(b *testing.B) {
	benchmark_dd("test-dd-2", 4194304, 512, b)
}

func Benchmark_dd_3(b *testing.B) {
	benchmark_dd("test-dd-3", 2097152, 1024, b)
}

func Benchmark_dd_4(b *testing.B) {
	benchmark_dd("test-dd-4", 1048576, 2048, b)
}

func benchmark_dd(of string, bs, count int, b *testing.B) {
	err := os.Remove(of)
	if !(err == nil || os.IsNotExist(err)) {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		err = dd(of, bs, count)
		if err != nil {
			b.Fatal(err)
		}

		err = os.Remove(of)
		if !(err == nil || os.IsNotExist(err)) {
			b.Fatal(err)
		}
	}
}
