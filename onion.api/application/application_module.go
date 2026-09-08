package application

type ApplicationSettings struct {
	PakNSaveEnrichBarcodes bool
}

type ApplicationModule struct{}

func Initialize(settings ApplicationSettings) (*ApplicationModule, error) {
	return &ApplicationModule{}, nil
}
