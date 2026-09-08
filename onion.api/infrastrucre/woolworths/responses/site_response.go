package responses

type SiteDetailItem struct {
	Site SiteInfo `json:"site"`
}

type SiteInfo struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	AddressLine1 string  `json:"addressLine1"`
	Suburb       string  `json:"suburb"`
	Postcode     string  `json:"postcode"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

type SiteResponse struct {
	SiteDetail []SiteDetailItem `json:"siteDetail"`
}
