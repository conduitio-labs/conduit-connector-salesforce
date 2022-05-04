package main

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
	sf "github.com/miquido/conduit-connector-salesforce"
	sfSource "github.com/miquido/conduit-connector-salesforce/source"
)

func main() {
	sdk.Serve(sf.Specification, sfSource.NewSource, nil)
}
