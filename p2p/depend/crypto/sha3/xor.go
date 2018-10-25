package sha3

var (
	xorIn            = xorInGeneric
	copyOut          = copyOutGeneric
	xorInUnaligned   = xorInGeneric
	copyOutUnaligned = copyOutGeneric
)

const xorImplementationUnaligned = "generic"
