// Package blocksutil provides utility functions for working
// with Blocks.
package blocksutil

import (
	blocks "gx/ipfs/QmSn9Td7xgxm9EV7iEjTckpUWmWApggzPxu7eFGWkkpwin/go-block-format"

	bl "github.com/Casper-dev/Casper-server/blocks"
)

// NewBlockGenerator returns an object capable of
// producing blocks.
func NewBlockGenerator() BlockGenerator {
	return BlockGenerator{}
}

// BlockGenerator generates BasicBlocks on demand.
// For each instace of BlockGenerator,
// each new block is different from the previous,
// although two different instances will produce the same.
type BlockGenerator struct {
	seq int
}

// Next generates a new BasicBlock.
func (bg *BlockGenerator) Next() *bl.BasicBlock {
	bg.seq++
	return bl.NewBlock([]byte(string(bg.seq)))
}

// Blocks generates as many BasicBlocks as specified by n.
func (bg *BlockGenerator) Blocks(n int) []blocks.Block {
	blocks := make([]blocks.Block, 0, n)
	for i := 0; i < n; i++ {
		b := bg.Next()
		blocks = append(blocks, b)
	}
	return blocks
}
