package responses

// ScrapedProductResponse is a normalized product from any retailer.
type ScrapedProductResponse struct {
	ExternalProductId string
	Name              string
	Brand             string
	Barcode           string
	BarcodeType       BarcodeType
	PriceCents        int
	SalePriceCents    *int
	HasPrice          bool
	UnitOfMeasure     string
	PackSize          string
	CategoryPath      []string
	ImageUrl          string
	InStock           *bool
}
