package mappers

import (
	"onion.api/infrastrucre/common/responses"
	"onion.api/persistence/entities"
)

func MapStoreChain(chain responses.StoreChain) entities.StoreChain {
	switch chain {
	case responses.StoreChainWoolworths:
		return entities.StoreChainWOOLWORTHS
	case responses.StoreChainPakNSave:
		return entities.StoreChainPAKNSAVE
	case responses.StoreChainNewWorld:
		return entities.StoreChainNEWWORLD
	default:
		return ""
	}
}
