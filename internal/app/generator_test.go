package app

import (
	"fmt"
	"testing"
)

func BenchmarkGenerateShortKey(b *testing.B) {
	g := Generate{}
	for i := 0; i < b.N; i++ {
		g.GenerateShortKey()
	}
}

func ExampleGenerate_GenerateShortKey() {
	g := Generate{}
	short := g.GenerateShortKey()
	fmt.Println(len(short))

	// Output:
	// 8
}
