package paknsave

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	common "onion.api/infrastrucre/common"
	"onion.api/infrastrucre/common/responses"
	paknsaveresponses "onion.api/infrastrucre/paknsave/responses"
	"onion.api/persistence/entities"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	sessionMintUrl   = "https://www.paknsave.co.nz/api/user/get-current-user"
	tokenRefreshSkew = 5 * time.Minute
)

type jwtPayload struct {
	Exp int64 `json:"exp"`
}

func isTokenValid(tokenString string) bool {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false
	}
	payloadBytes, decodeError := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeError != nil {
		return false
	}
	var payload jwtPayload
	if unmarshalError := json.Unmarshal(payloadBytes, &payload); unmarshalError != nil {
		return false
	}
	if payload.Exp == 0 {
		return true
	}
	return time.Now().Unix() < payload.Exp-int64(tokenRefreshSkew.Seconds())
}

type PakNSaveClient struct {
	*common.BaseRetailer
	storeName    string
	currentToken string
	tokenExpiry  time.Time
}

func NewPakNSaveClient(retailer *common.BaseRetailer, storeName string) *PakNSaveClient {
	return &PakNSaveClient{
		BaseRetailer: retailer,
		storeName:    storeName,
	}
}

func (client *PakNSaveClient) getSession(requestContext context.Context) (string, error) {
	if client.currentToken != "" && time.Now().Before(client.tokenExpiry) {
		return client.currentToken, nil
	}

	liveSession, fetchError := client.Queries().FindLiveRetailerSession(requestContext, entities.FindLiveRetailerSessionParams{
		StoreChain: entities.StoreChainPAKNSAVE,
		Now:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if fetchError == nil && liveSession.Token != "" && isTokenValid(liveSession.Token) {
		client.currentToken = liveSession.Token
		if liveSession.ExpiresAtUtc.Valid {
			client.tokenExpiry = liveSession.ExpiresAtUtc.Time
		} else {
			client.tokenExpiry = time.Now().Add(30 * time.Minute)
		}
		return client.currentToken, nil
	}

	return client.mintToken(requestContext)
}

func (client *PakNSaveClient) mintToken(requestContext context.Context) (string, error) {
	results, executeError := client.Execute(requestContext, []common.RetailClientRequest{
		{
			Method:  http.MethodPost,
			Path:    sessionMintUrl,
			Label:   "paknsave mint",
			Headers: map[string][]string{"Content-Type": {"application/json"}},
		},
	})
	if executeError != nil {
		return "", fmt.Errorf("paknsave: mint request failed: %w", executeError)
	}
	if !results[0].IsOk() {
		return "", fmt.Errorf("paknsave: mint request failed: %v", results[0].Error)
	}
	response := results[0].Response

	var mintResponse paknsaveresponses.MintResponse
	if unmarshalError := json.Unmarshal(response.Body, &mintResponse); unmarshalError != nil {
		return "", fmt.Errorf("paknsave: decode mint response: %w", unmarshalError)
	}
	if mintResponse.AccessToken == "" {
		return "", fmt.Errorf("paknsave: mint response missing access_token")
	}

	var expiresAt pgtype.Timestamptz
	if mintResponse.ExpiresTime != "" {
		if parsedTime, parseError := time.Parse(time.RFC3339, mintResponse.ExpiresTime); parseError == nil {
			expiresAt = pgtype.Timestamptz{Time: parsedTime, Valid: true}
		}
	}

	_, upsertError := client.Queries().UpsertRetailerSession(requestContext, entities.UpsertRetailerSessionParams{
		StoreChain:   entities.StoreChainPAKNSAVE,
		Token:        mintResponse.AccessToken,
		ExpiresAtUtc: expiresAt,
	})
	if upsertError != nil {
		return "", fmt.Errorf("paknsave: persist session: %w", upsertError)
	}

	client.currentToken = mintResponse.AccessToken
	if expiresAt.Valid {
		client.tokenExpiry = expiresAt.Time
	} else {
		client.tokenExpiry = time.Now().Add(30 * time.Minute)
	}

	return client.currentToken, nil
}

func (client *PakNSaveClient) GetStores(requestContext context.Context) ([]responses.StoreResponse, error) {
	token, sessionError := client.getSession(requestContext)
	if sessionError != nil {
		return nil, responses.NewScrapingExceptionResponse("paknsave: get session", sessionError)
	}

	results, executeError := client.Execute(requestContext, []common.RetailClientRequest{
		{
			Method: "GET",
			Path:   "/store",
			Label:  "paknsave stores",
			Headers: map[string][]string{
				"Authorization": {fmt.Sprintf("Bearer %s", token)},
			},
		},
	})
	if executeError != nil {
		return nil, responses.NewScrapingExceptionResponse("paknsave: fetch stores", executeError)
	}
	if !results[0].IsOk() {
		return nil, responses.NewScrapingExceptionResponse("paknsave: fetch stores", results[0].Error)
	}
	storeList, deserializeError := DeserializeStoreList(results[0].Response.Body, "paknsave: decode stores")
	if deserializeError != nil {
		return nil, deserializeError
	}
	var stores []responses.StoreResponse
	for _, store := range storeList.Stores {
		stores = append(stores, responses.StoreResponse{
			Retailer:  responses.StoreChainPakNSave,
			Id:        store.Id,
			Name:      store.Name,
			Address:   store.Address,
			Latitude:  store.Latitude,
			Longitude: store.Longitude,
		})
	}
	return stores, nil
}

func (client *PakNSaveClient) GetProducts(requestContext context.Context) ([]responses.ScrapedProductResponse, error) {
	return nil, nil
}
