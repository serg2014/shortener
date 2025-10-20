package app

import "fmt"

func ExampleGenerate_GenerateShortKey() {
	g := Generate{}
	short := g.GenerateShortKey()
	fmt.Println(len(short))

	// Output:
	// 8
}
