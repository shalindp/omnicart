package infrastrucre

type InfrastructureSettings struct {
	WoolworthsReferenceStore string
	PakNSaveReferenceStore   string
}

type InfrastructureModule struct{}

func Initialize(settings InfrastructureSettings) (*InfrastructureModule, error) {
	return &InfrastructureModule{}, nil
}
