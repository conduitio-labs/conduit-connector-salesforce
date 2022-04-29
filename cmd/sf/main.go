package main

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/miquido/conduit-connector-salesforce/internal"
	"github.com/miquido/conduit-connector-salesforce/source"
)

func main() {
	sdk.Serve(internal.Specification, source.NewSource, nil)
}
