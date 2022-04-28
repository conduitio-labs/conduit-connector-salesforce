package response

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	InstanceUrl string `json:"instance_url"`
	Id          string `json:"id"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`
}

func (t TokenResponse) GetAccessToken() string {
	return t.AccessToken
}

func (t TokenResponse) GetInstanceUrl() string {
	return t.InstanceUrl
}
