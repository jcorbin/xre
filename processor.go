package xre

// Processor represents a piece of structure processing logic. Process gets
// called for each piece of matched sub-structure within some level of
// structure. The last flag indicates whether this is the last piece of
// sub-structure. After Process has been called with last=true, it may be
// called again to start processing the next (semantically sibling) structure
// to the one just ended.
//
// If a Processor also implements io.ReaderFrom, then it can be used as a
// toplevel processor; without such a toplevel processor, the Environment must
// provide default stream extraction semantics.
type Processor interface {
	Process(buf []byte, last bool) error
}

// ProtoProcessor is a nearly constructed Processor. Useful for constructing
// generic Command implementations, to encapsulate some piece of processing
// that only needs to know the next step (doesn't need to control the creation
// of the next step, and doesn't need Environment access).
type ProtoProcessor interface {
	Create(next Processor) Processor
}
