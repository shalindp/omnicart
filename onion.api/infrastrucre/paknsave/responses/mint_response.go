package responses

type MintResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresTime string `json:"expires_time"`
	UserTicket  string `json:"userTicket,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}
