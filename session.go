package session

import (
	"container/list"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ProjectRoot, _ = os.Getwd()
	SessManager    *SessionManager
	ConfigLocation = filepath.Join(ProjectRoot, "configs", "session.json")
)

type (
	dict     = map[interface{}]interface{}
	sessDict = map[string]*list.Element
)

type Session struct {
	sessionId    string
	lastAccessed time.Time
	value        dict
	lock         sync.RWMutex
}

type SessionManagerConfig struct {
	CookieName       string `json:"cookie_name"`
	CleanerInterval  int64  `json:"cleaner_interval"`
	MaxLifetime      int64  `json:"max_lifetime"`
	HTTPOnly         bool   `json:"http_only"`
	Secure           bool   `json:"secure"`
	CookieLifetime   int    `json:"cookie_lifetime"`
	Domain           string `json:"domain"`
	EnableHttpHeader bool   `json:"enable_http_header"`
	SessionHeader    string `json:"session_header"`
}

type SessionManager struct {
	lock     sync.RWMutex
	sessions sessDict
	list     *list.List
	Config   *SessionManagerConfig
}

func GetManagerConfig() *SessionManagerConfig {
	confFile, err := ioutil.ReadFile(ConfigLocation)
	if err != nil {
		return nil
	}

	config := SessionManagerConfig{}

	if err := json.Unmarshal([]byte(confFile), &config); err != nil {
		return nil
	}

	return &config
}

func (s *Session) Get(key interface{}) interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if val, ok := s.value[key]; ok {
		return val
	}

	return nil
}

func (s *Session) Exist(key interface{}) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.value[key]; ok {
		return true
	}

	return false
}

func (s *Session) Set(key, value interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.value[key] = value

	return nil
}

func (s *Session) Delete(key interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.value, key)

	return nil
}

func NewSessionManager(conf *SessionManagerConfig) *SessionManager {
	if conf.MaxLifetime == 0 {
		conf.MaxLifetime = conf.CleanerInterval
	}

	return &SessionManager{
		list:     list.New(),
		sessions: make(sessDict),
		Config:   conf,
	}
}

func (sm *SessionManager) GetSessionId(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sm.Config.CookieName)

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

func (sm *SessionManager) SessionExist(sid string) bool {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	if _, ok := sm.sessions[sid]; ok {
		return true
	}

	return false
}

func (sm *SessionManager) SessionReadOrCreate(r *http.Request) (*Session, error) {
	sid, err := sm.GetSessionId(r)
	if err != nil || sid == "" {
		return nil, err
	}

	sm.lock.RLock()
	if ele, ok := sm.sessions[sid]; ok {
		go sm.SessionUpdate(sid)
		sm.lock.RUnlock()

		return ele.Value.(*Session), nil
	}

	sm.lock.RUnlock()
	sm.lock.Lock()
	newSess := &Session{
		sessionId:    sid,
		lastAccessed: time.Now(),
		value:        make(dict),
	}
	ele := sm.list.PushFront(newSess)
	sm.sessions[sid] = ele
	sm.lock.Unlock()

	return newSess, nil
}

func (sm *SessionManager) SessionRead(r *http.Request) (*Session, error) {
	sid, err := sm.GetSessionId(r)
	if err != nil || sid == "" {
		return nil, err
	}

	sm.lock.RLock()
	if ele, ok := sm.sessions[sid]; ok {
		go sm.SessionUpdate(sid)
		sm.lock.RUnlock()

		return ele.Value.(*Session), nil
	}

	return nil, errors.New("Session not found")
}

func (sm *SessionManager) SessionUpdate(sid string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if ele, ok := sm.sessions[sid]; ok {
		ele.Value.(*Session).lastAccessed = time.Now()
		sm.list.Init().MoveToFront(ele)
		return nil
	}

	return errors.New("Error while updating session")
}

func (sm *SessionManager) SessionDestroy(sid string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if ele, ok := sm.sessions[sid]; ok {
		delete(sm.sessions, sid)
		sm.list.Remove(ele)
		return nil
	}

	return errors.New("Error while deleting session")
}

func (sm *SessionManager) GlobalCleaner() {
	sm.lock.RLock()
	for {
		ele := sm.list.Back()
		if ele == nil {
			break
		}

		if (ele.Value.(*Session).lastAccessed.Unix() + sm.Config.MaxLifetime) < time.Now().Unix() {
			sm.lock.RUnlock()
			sm.lock.Lock()
			delete(sm.sessions, ele.Value.(*Session).sessionId)
			sm.list.Remove(ele)
			sm.lock.Unlock()
			sm.lock.RLock()
		} else {
			break
		}
	}
	sm.lock.RUnlock()
	time.AfterFunc(time.Duration(sm.Config.CleanerInterval)*time.Second, func() { sm.GlobalCleaner() })
}

func (sm *SessionManager) SessionRefresh(oldSid, sid string) (*Session, error) {
	sm.lock.RLock()
	if ele, ok := sm.sessions[oldSid]; ok {
		go sm.SessionUpdate(oldSid)
		sm.lock.RUnlock()
		sm.lock.Lock()
		ele.Value.(*Session).sessionId = sid
		sm.sessions[sid] = ele
		delete(sm.sessions, oldSid)
		sm.lock.Unlock()

		return ele.Value.(*Session), nil
	}
	sm.lock.RUnlock()
	sm.lock.Lock()
	newSess := &Session{
		sessionId:    sid,
		lastAccessed: time.Now(),
		value:        make(dict),
	}
	ele := sm.list.Init().PushFront(newSess)
	sm.sessions[sid] = ele
	sm.lock.Unlock()

	return newSess, nil
}

func (sm *SessionManager) SessionCount() int {
	return sm.list.Len()
}

func init() {
	SessManager = NewSessionManager(GetManagerConfig())
	go SessManager.GlobalCleaner()
}
