package main

import (
    "fmt"
    "github.com/pascal910107/idgen"
)

func main() {
    g, _ := idgen.NewGenerator(1, 42) // region=1, node=42
    id, _ := g.Next()
    fmt.Println("ID (Hex):", id)
    fmt.Println("ID (Base64):", id.Base64URL())

    // Decode for debugging
    ep, ts, r, n, seq := id.Decode()
    fmt.Printf("epoch=%d ts(ms)=%d region=%d node=%d seq=%d\n", ep, ts, r, n, seq)
}
