package main

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/miquido/conduit-connector-salesforce"
	"github.com/miquido/conduit-connector-salesforce/source"
)

func main() {
	sdk.Serve(salesforce.Specification, source.NewSource, nil)
}
