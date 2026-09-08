package responses

type StoreListResponse struct {
	Stores []PakNSaveStore `json:"stores"`
}

type PakNSaveStore struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Region       string  `json:"region,omitempty"`
	Address      string  `json:"address,omitempty"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	OnlineActive bool    `json:"onlineActive"`
}
