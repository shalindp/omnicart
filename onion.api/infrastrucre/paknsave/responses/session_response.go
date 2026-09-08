package responses

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	common "onion.api/infrastrucre/common"
	commonresponses "onion.api/infrastrucre/common/responses"
)

type Session struct {
	Token     string
	ExpiresAt *time.Time
}

func (session Session) IsValid(currentTime time.Time) bool {
	if session.Token == "" {
		return false
	}
	if session.ExpiresAt == nil {
		return true
	}
	return currentTime.Before(session.ExpiresAt.Add(-RefreshSkew))
}

var SessionNone = Session{}

const (
	SessionMintPath = "https://www.paknsave.co.nz/api/user/get-current-user"
	RefreshSkew     = 5 * time.Minute
)

type PakNSaveSessionProvider struct {
	store   common.SessionStore
	mutex   sync.Mutex
	current Session
	primed  bool
}

func NewPakNSaveSessionProvider(store common.SessionStore) *PakNSaveSessionProvider {
	return &PakNSaveSessionProvider{store: store}
}

func (provider *PakNSaveSessionProvider) Prime() error {
	if provider.store == nil {
		return nil
	}
	provider.mutex.Lock()
	if provider.primed || provider.current.Token != "" {
		provider.mutex.Unlock()
		return nil
	}
	provider.primed = true
	provider.mutex.Unlock()

	storedSession, loadError := provider.store.Load(context.Background(), commonresponses.StoreChainPakNSave)
	if loadError != nil || storedSession == nil {
		return nil
	}
	provider.mutex.Lock()
	if provider.current.Token == "" {
		provider.current = Session{Token: storedSession.Token, ExpiresAt: storedSession.ExpiresAt}
	}
	provider.mutex.Unlock()
	return nil
}

func (provider *PakNSaveSessionProvider) MintRequest() common.RetailClientRequest {
	return common.RetailClientRequest{
		Method: "POST", Path: SessionMintPath, Label: "paknsave mint session",
		Headers: map[string][]string{"Content-Type": {"application/json"}},
	}
}

func (provider *PakNSaveSessionProvider) TryAccept(response common.RetailClientResponse) bool {
	var mintResponse MintResponse
	if unmarshalError := json.Unmarshal(response.Body, &mintResponse); unmarshalError != nil {
		return false
	}
	if mintResponse.AccessToken == "" {
		return false
	}
	var expiresAt *time.Time
	if mintResponse.ExpiresTime != "" {
		if parsedTime, parseError := time.Parse(time.RFC3339, mintResponse.ExpiresTime); parseError == nil {
			expiresAt = &parsedTime
		}
	}
	provider.mutex.Lock()
	provider.current = Session{Token: mintResponse.AccessToken, ExpiresAt: expiresAt}
	provider.mutex.Unlock()
	go provider.persist(common.StoredSession{Token: mintResponse.AccessToken, ExpiresAt: expiresAt})
	return true
}

func (provider *PakNSaveSessionProvider) persist(session common.StoredSession) {
	if provider.store == nil {
		return
	}
	provider.store.Save(context.Background(), commonresponses.StoreChainPakNSave, session)
}

func (provider *PakNSaveSessionProvider) Headers(currentTime time.Time) map[string][]string {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()
	if provider.current.IsValid(currentTime) {
		return map[string][]string{"Authorization": {fmt.Sprintf("Bearer %s", provider.current.Token)}}
	}
	return nil
}

func (provider *PakNSaveSessionProvider) IsExpired(response common.RetailClientResponse) bool {
	return response.StatusCode == 401 || response.StatusCode == 403
}

func (provider *PakNSaveSessionProvider) Invalidate() {
	provider.mutex.Lock()
	provider.current = SessionNone
	provider.primed = true
	provider.mutex.Unlock()
	go provider.clear()
}

func (provider *PakNSaveSessionProvider) clear() {
	if provider.store == nil {
		return
	}
	provider.store.Clear(context.Background(), commonresponses.StoreChainPakNSave)
}

func (provider *PakNSaveSessionProvider) Token() string {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()
	return provider.current.Token
}

var _ common.RetailSession = (*PakNSaveSessionProvider)(nil)
