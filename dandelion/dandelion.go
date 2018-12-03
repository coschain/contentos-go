package dandelion

type Dandelion interface {
	OpenDatabase()

	GenerateBlock()

	//GenerateBlocks(count uint32)
	//
	//// deadline
	//GenerateBlockUntil(timestamp uint64)
	//
	//// pass by time
	//GenerateBlockFor(timestamp uint64)
	//
	//Validate()

	Clean()
}
