package main

import "io"

// processor represents a piece of structure processing logic. Process gets
// called for each piece of matched sub-structure within some level of
// structure. The last flag indicates whether this is the last piece of
// sub-structure. After Process has been called with last=true, it may be
// called again to start processing the next (semantically sibling) structure
// to the one just ended.
type processor interface {
	Process(buf []byte, last bool) error
}

// processorIO is a processor that supports streaming. Such a processor may be
// used as a top level processor starting out a command chain; all other
// processors must be wrapped/driven by a processorIO.
type processorIO interface {
	processor
	io.ReaderFrom
}
