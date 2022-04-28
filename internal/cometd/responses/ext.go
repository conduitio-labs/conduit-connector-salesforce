package responses

// ext is an arbitrary values that allow extensions to be negotiated and implemented between server and client implementations.
// See: https://docs.cometd.org/current7/reference/#_bayeux_ext
type ext map[string]interface{}
