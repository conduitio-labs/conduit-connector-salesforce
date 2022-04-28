package response

type FailureResponseError struct {
	ErrorName        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (f FailureResponseError) Error() string {
	return f.ErrorDescription
}
