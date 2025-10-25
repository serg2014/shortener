package app

import "fmt"

func ExampleGenerate_GenerateShortKey() {
	g := Generate{}
	short, err := g.GenerateShortKey()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(short))

	// Output:
	// 8
}
