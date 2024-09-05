package session

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type (
	dict     = map[interface{}]interface{}
	sessDict = map[string]*Session
)

type Session struct {
	sessionId    string
	lastAccessed time.Time
	sd           dict
	lock         sync.RWMutex
}

func (s *Session) Get(key interface{}) interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if val, ok := s.sd[key]; ok {
		return val
	}

	return nil
}

func (s *Session) Exist(key interface{}) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.sd[key]; ok {
		return true
	}

	return false
}

func (s *Session) Set(key, sd interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.sd[key] = sd

	return nil
}

func (s *Session) Delete(key interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.sd, key)

	return nil
}

type SessionCookie struct {
	Name     string
	Domain   string
	HTTPOnly bool
	Secure   bool
	Lifetime time.Duration
}

type SessionManagerConfig struct {
	CleanerInterval    time.Duration
	MaxLifetime        time.Duration
	CookieLifetime     time.Duration
	EnableHttpHeader   bool
	SessionHeader      string
	AutoRefreshSession bool
}

type SessionManager struct {
	lock     sync.RWMutex
	sessions sessDict
	Config   SessionManagerConfig
	Cookie   SessionCookie
}

func (sm *SessionManager) GetSessionId(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sm.Cookie.Name)

	if err != nil || cookie.Value == "" {

		if sm.Config.EnableHttpHeader {
			sids, found := r.Header[sm.Config.SessionHeader]
			if found && len(sids) != 0 {
				return sids[0], nil
			}
		}

		return "", err
	}

	return url.QueryUnescape(cookie.Value)
}

func (sm *SessionManager) GetSessionIdFromHeader(r *http.Request) (string, error) {
	if sm.Config.EnableHttpHeader {
		sids, found := r.Header[sm.Config.SessionHeader]
		if found && len(sids) != 0 {
			return sids[0], nil
		}
	}

	return "", fmt.Errorf("error getting session id from %s", sm.Config.SessionHeader)
}

func (sm *SessionManager) GetSessionIdFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sm.Cookie.Name)

	if err != nil || cookie.Value == "" {
		return "", err
	}

	return url.QueryUnescape(cookie.Value)
}

func (sm *SessionManager) ListSessions() {
	sm.lock.RLock()
	for sid, s := range sm.sessions {
		if s == nil {
			continue
		}
		log.Printf("SID : %s, Session Data : %v", sid, s.sd)
	}
	sm.lock.RUnlock()
}

func (sm *SessionManager) SessionCount() int {
	return len(sm.sessions)
}

func (sm *SessionManager) SessionRefresh(oldSid, sid string) (*Session, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if s, ok := sm.sessions[oldSid]; ok {
		s.sessionId = sid
		sm.sessions[sid] = s
		delete(sm.sessions, oldSid)

		return s, nil
	}
	newSess := &Session{
		sessionId:    sid,
		lastAccessed: time.Now(),
		sd:           make(dict),
	}
	sm.sessions[sid] = newSess

	return newSess, nil
}

func (sm *SessionManager) SessionExist(sid string) bool {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	_, ok := sm.sessions[sid]
	return ok
}

// Update the session access time. Refresh Session
func (sm *SessionManager) SessionUpdate(sid string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if s, ok := sm.sessions[sid]; ok {
		s.lastAccessed = time.Now()
		return nil
	}

	return errors.New("error while updating session")
}

// Remove the session for matching sid
func (sm *SessionManager) SessionDestroy(sid string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if _, ok := sm.sessions[sid]; ok {
		delete(sm.sessions, sid)
		return nil
	}

	return errors.New("error while deleting session")
}

// Read session. Error out if session not found
func (sm *SessionManager) SessionRead(r *http.Request) (*Session, error) {
	sid, err := sm.GetSessionId(r)
	if err != nil || sid == "" {
		return nil, err
	}

	sm.lock.RLock()
	defer sm.lock.RUnlock()
	if s, ok := sm.sessions[sid]; ok {
		if sm.Config.AutoRefreshSession {
			go sm.SessionUpdate(sid)
		}
		return s, nil
	}

	return nil, errors.New("session not found")
}

func (sm *SessionManager) SessionCreate(sid string) (*Session, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	s := &Session{
		sessionId:    sid,
		lastAccessed: time.Now(),
		sd:           make(dict),
	}
	sm.sessions[sid] = s

	return s, nil
}

func (sm *SessionManager) GlobalCleaner() {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	for sid, s := range sm.sessions {
		if s == nil {
			continue
		}

		if time.Now().After(s.lastAccessed.Add(sm.Config.MaxLifetime)) {
			delete(sm.sessions, sid)
		}
	}
	time.AfterFunc(sm.Config.CleanerInterval, func() { sm.GlobalCleaner() })
}

// Create a new instance of session manager.
func New() *SessionManager {

	sm := &SessionManager{
		sessions: make(sessDict),
		Config: SessionManagerConfig{
			CleanerInterval:    1 * time.Minute,
			MaxLifetime:        24 * time.Hour,
			EnableHttpHeader:   false,
			SessionHeader:      "",
			AutoRefreshSession: false,
		},
		Cookie: SessionCookie{
			Name:     "sessionid",
			Domain:   "",
			HTTPOnly: true,
			Secure:   false,
			Lifetime: 24 * time.Hour,
		},
	}

	go sm.GlobalCleaner()

	return sm
}
