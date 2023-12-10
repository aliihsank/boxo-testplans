package main

import (
	"math/rand"

	block "github.com/ipfs/go-block-format"
)

func generateBlocksOfSize(n int, size uint64, r *rand.Rand) []block.Block {
	generatedBlocks := make([]block.Block, 0, n)
	for i := 0; i < n; i++ {
		// rand.Read never errors
		buf := make([]byte, size)
		r.Read(buf)
		b := block.NewBlock(buf)
		generatedBlocks = append(generatedBlocks, b)
	}
	return generatedBlocks
}
