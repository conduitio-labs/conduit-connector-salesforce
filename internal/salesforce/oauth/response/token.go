package response

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	ID          string `json:"id"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`
}

func (t TokenResponse) GetAccessToken() string {
	return t.AccessToken
}

func (t TokenResponse) GetInstanceURL() string {
	return t.InstanceURL
}
