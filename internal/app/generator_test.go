package app

import "testing"

func BenchmarkGenerateShortKey(b *testing.B) {
	g := Generate{}
	for i := 0; i < b.N; i++ {
		g.GenerateShortKey()
	}
}
