package responses

// StoreChain represents a retail chain.
type StoreChain string

const (
	StoreChainWoolworths StoreChain = "WOOLWORTHS"
	StoreChainPakNSave   StoreChain = "PAKNSAVE"
	StoreChainNewWorld   StoreChain = "NEW_WORLD"
)

func (sc StoreChain) String() string {
	return string(sc)
}

// StoreResponse is a store returned by the GetStores query.
type StoreResponse struct {
	Retailer   StoreChain `json:"retailer"`
	Id         string     `json:"id,omitempty"`
	StoreChain StoreChain `json:"store_chain"`
	Name       string     `json:"name"`
	Region     string     `json:"region"`
	Address    string     `json:"address,omitempty"`
	Latitude   float64    `json:"latitude,omitempty"`
	Longitude  float64    `json:"longitude,omitempty"`
}
