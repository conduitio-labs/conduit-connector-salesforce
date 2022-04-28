package requests

import "encoding/json"

type Request interface {
	json.Marshaler
}
