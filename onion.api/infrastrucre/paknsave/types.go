package paknsave

import (
	"encoding/json"
	"fmt"

	paknsaveresponses "onion.api/infrastrucre/paknsave/responses"
)

const ApiBaseUrl = "https://api-prod.paknsave.co.nz/v1/edge"

type PakNSaveStore struct {
	Id        string  `json:"id"`
	Name      string  `json:"name"`
	Region    string  `json:"region,omitempty"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Online    bool    `json:"-"`
}

type StoreListResponse struct {
	Stores []PakNSaveStore `json:"stores"`
}

func DeserializeStoreList(body []byte, context string) (*StoreListResponse, error) {
	var apiResponse paknsaveresponses.StoreListResponse
	if unmarshalError := json.Unmarshal(body, &apiResponse); unmarshalError != nil {
		return nil, fmt.Errorf("%s: %w", context, unmarshalError)
	}
	var stores []PakNSaveStore
	for _, store := range apiResponse.Stores {
		if store.ID == "" {
			continue
		}
		stores = append(stores, PakNSaveStore{
			Id: store.ID, Name: store.Name, Address: store.Address, Region: store.Region,
			Latitude: store.Latitude, Longitude: store.Longitude, Online: store.OnlineActive,
		})
	}
	return &StoreListResponse{Stores: stores}, nil
}
