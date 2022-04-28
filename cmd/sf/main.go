package main

import (
	"github.com/conduitio/conduit-connector-salesforce/internal"
	"github.com/conduitio/conduit-connector-salesforce/source"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

func main() {
	sdk.Serve(internal.Specification, source.NewSource, nil)
}
