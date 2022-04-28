package response

type FailureResponse struct {
	ErrorName        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (f FailureResponse) Error() string {
	return f.ErrorDescription
}
