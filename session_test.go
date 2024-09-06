package session

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSession_Get(t *testing.T) {
	// Case 1: Key Exists
	s := &Session{sd: make(dict)}
	s.Set("key1", "value1")

	if val := s.Get("key1"); val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// Case 2: Key Does Not Exist
	if val := s.Get("key2"); val != nil {
		t.Errorf("Expected nil, got %v", val)
	}

	// Case 3: Empty Session
	sEmpty := &Session{sd: make(dict)}
	if val := sEmpty.Get("key1"); val != nil {
		t.Errorf("Expected nil, got %v", val)
	}

	// Case 4: Nil Key
	if val := s.Get(nil); val != nil {
		t.Errorf("Expected nil, got %v", val)
	}

	// Case 5: Concurrent Access
	sConcurrent := &Session{sd: make(dict)}
	sConcurrent.Set("key1", "value1")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if val := sConcurrent.Get("key1"); val != "value1" {
				t.Errorf("Expected value1, got %v", val)
			}
		}()
	}
	wg.Wait()
}

func TestSession_Exist(t *testing.T) {
	// Case 1: Key Exists
	s := &Session{sd: make(dict)}
	s.Set("key1", "value1")

	if !s.Exist("key1") {
		t.Errorf("Expected key1 to exist")
	}

	// Case 2: Key Does Not Exist
	if s.Exist("key2") {
		t.Errorf("Expected key2 to not exist")
	}

	// Case 3: Empty Session
	sEmpty := &Session{sd: make(dict)}
	if sEmpty.Exist("key1") {
		t.Errorf("Expected key1 to not exist in empty session")
	}

	// Case 4: Nil Key
	if s.Exist(nil) {
		t.Errorf("Expected nil key to not exist")
	}

	// Case 5: Concurrent Access
	sConcurrent := &Session{sd: make(dict)}
	sConcurrent.Set("key1", "value1")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !sConcurrent.Exist("key1") {
				t.Errorf("Expected key1 to exist")
			}
		}()
	}
	wg.Wait()
}

func TestSession_Set(t *testing.T) {
	// Case 1: Set New Key-Value Pair
	s := &Session{sd: make(dict)}
	s.Set("key1", "value1")

	if val := s.Get("key1"); val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// Case 2: Update Existing Key
	s.Set("key1", "value2")
	if val := s.Get("key1"); val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}

	// Case 3: Set Nil Value
	s.Set("key2", nil)
	if val := s.Get("key2"); val != nil {
		t.Errorf("Expected nil, got %v", val)
	}

	// Case 4: Set Nil Key
	s.Set(nil, "value3")
	if val := s.Get(nil); val != "value3" {
		t.Errorf("Expected value3, got %v", val)
	}

	// Case 5: Concurrent Set Operations
	sConcurrent := &Session{sd: make(dict)}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)
			sConcurrent.Set(key, value)
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		expectedValue := fmt.Sprintf("value%d", i)
		if val := sConcurrent.Get(key); val != expectedValue {
			t.Errorf("Expected %v, got %v", expectedValue, val)
		}
	}
}

func TestSession_Delete(t *testing.T) {
	// Case 1: Delete Existing Key
	s := &Session{sd: make(dict)}
	s.Set("key1", "value1")
	s.Delete("key1")

	if val := s.Get("key1"); val != nil {
		t.Errorf("Expected nil, got %v", val)
	}

	// Case 2: Delete Non-Existent Key
	s.Set("key2", "value2")
	s.Delete("key3") // key3 does not exist
	if val := s.Get("key2"); val != "value2" {
		t.Errorf("Expected value2, got %v", val)
	}

	// Case 3: Delete From Empty Session
	sEmpty := &Session{sd: make(dict)}
	sEmpty.Delete("key1") // key1 does not exist in empty session
	if val := sEmpty.Get("key1"); val != nil {
		t.Errorf("Expected nil, got %v", val)
	}

	// Case 4: Delete Nil Key
	s.Set(nil, "value3")
	s.Delete(nil)
	if val := s.Get(nil); val != nil {
		t.Errorf("Expected nil, got %v", val)
	}

	// Case 5: Concurrent Delete Operations
	sConcurrent := &Session{sd: make(dict)}
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		sConcurrent.Set(key, value)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			sConcurrent.Delete(key)
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		if val := sConcurrent.Get(key); val != nil {
			t.Errorf("Expected nil, got %v", val)
		}
	}
}

func TestSessionManager_GetSessionId(t *testing.T) {
	sm := New()

	// Case 1: Session ID from Cookie
	req := httptest.NewRequest("GET", "/", nil)
	cookie := &http.Cookie{Name: sm.Cookie.Name, Value: "sessionid123"}
	req.AddCookie(cookie)

	sid, err := sm.GetSessionId(req)
	if err != nil || sid != "sessionid123" {
		t.Errorf("Expected sessionid123, got %v, error: %v", sid, err)
	}

	// Case 2: Session ID from Header
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Session-Id", "header-sessionid123")
	sm.Config.EnableHttpHeader = true
	sm.Config.SessionHeader = "Session-Id"

	sid, err = sm.GetSessionId(req)
	if err != nil || sid != "header-sessionid123" {
		t.Errorf("Expected header-sessionid123, got %v, error: %v", sid, err)
	}

	// Case 3: No Session ID in Cookie or Header
	req = httptest.NewRequest("GET", "/", nil)
	sm.Config.EnableHttpHeader = false
	sm.Config.SessionHeader = "Session-Id"

	sid, err = sm.GetSessionId(req)
	if err == nil || sid != "" {
		t.Errorf("Expected empty session ID, got %v, error: %v", sid, err)
	}

	// Case 4: Invalid Cookie Value
	req = httptest.NewRequest("GET", "/", nil)
	cookie = &http.Cookie{Name: sm.Cookie.Name, Value: "%"}
	req.AddCookie(cookie)

	sid, err = sm.GetSessionId(req)
	if err == nil || sid != "" {
		t.Errorf("Expected error and empty session ID, got %v, error: %v", sid, err)
	}

	// Case 5: Header Disabled
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Session-Id", "header-sessionid123")
	sm.Config.EnableHttpHeader = false

	sid, err = sm.GetSessionId(req)
	if err == nil || sid != "" {
		t.Errorf("Expected empty session ID, got %v, error: %v", sid, err)
	}
}

func TestSessionManager_GetSessionIdFromHeader(t *testing.T) {
	sm := New()
	sm.Config.EnableHttpHeader = true
	sm.Config.SessionHeader = "Session-Id"

	// Case 1: Session ID from Header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(sm.Config.SessionHeader, "header-sessionid123")

	sid, err := sm.GetSessionIdFromHeader(req)
	if err != nil || sid != "header-sessionid123" {
		t.Errorf("Expected header-sessionid123, got %v, error: %v", sid, err)
	}

	// Case 2: Header Not Present
	req = httptest.NewRequest("GET", "/", nil)

	sid, err = sm.GetSessionIdFromHeader(req)
	if err == nil || sid != "" {
		t.Errorf("Expected error and empty session ID, got %v, error: %v", sid, err)
	}

	// Case 3: Header Disabled
	sm.Config.EnableHttpHeader = false
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set(sm.Config.SessionHeader, "header-sessionid123")

	sid, err = sm.GetSessionIdFromHeader(req)
	if err == nil || sid != "" {
		t.Errorf("Expected error and empty session ID, got %v, error: %v", sid, err)
	}

	// Case 4: Empty Header Value
	sm.Config.EnableHttpHeader = true
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set(sm.Config.SessionHeader, "")

	sid, err = sm.GetSessionIdFromHeader(req)
	if err == nil || sid != "" {
		t.Errorf("Expected error and empty session ID, got %v, error: %v", sid, err)
	}
}

func TestSessionManager_GetSessionIdFromCookie(t *testing.T) {
	sm := New()

	// Case 1: Session ID from Cookie
	req := httptest.NewRequest("GET", "/", nil)
	cookie := &http.Cookie{Name: sm.Cookie.Name, Value: "sessionid123"}
	req.AddCookie(cookie)

	sid, err := sm.GetSessionIdFromCookie(req)
	if err != nil || sid != "sessionid123" {
		t.Errorf("Expected sessionid123, got %v, error: %v", sid, err)
	}

	// Case 2: Cookie Not Present
	req = httptest.NewRequest("GET", "/", nil)

	sid, err = sm.GetSessionIdFromCookie(req)
	if err == nil || sid != "" {
		t.Errorf("Expected error and empty session ID, got %v, error: %v", sid, err)
	}

	// Case 3: Empty Cookie Value
	req = httptest.NewRequest("GET", "/", nil)
	cookie = &http.Cookie{Name: sm.Cookie.Name, Value: ""}
	req.AddCookie(cookie)

	sid, err = sm.GetSessionIdFromCookie(req)
	if err == nil || sid != "" {
		t.Fatalf("Expected error and empty session ID, got %v, error: %v", sid, err)
	}

	// Case 4: Invalid Cookie Value
	req = httptest.NewRequest("GET", "/", nil)
	cookie = &http.Cookie{Name: sm.Cookie.Name, Value: "%"}
	req.AddCookie(cookie)

	sid, err = sm.GetSessionIdFromCookie(req)
	if err == nil || sid != "" {
		t.Errorf("Expected error and empty session ID, got %v, error: %v", sid, err)
	}
}

func TestSessionManager_SessionCount(t *testing.T) {
	// Case 1: Count with Multiple Sessions
	sm := New()
	sm.SessionCreate("sessionid123")
	sm.SessionCreate("sessionid456")

	count := sm.SessionCount()
	if count != 2 {
		t.Errorf("Expected 2, got %v", count)
	}

	// Case 2: Count with No Sessions
	smEmpty := New()
	count = smEmpty.SessionCount()
	if count != 0 {
		t.Errorf("Expected 0, got %v", count)
	}

	// Case 3: Count with One Session
	smOne := New()
	smOne.SessionCreate("sessionid123")
	count = smOne.SessionCount()
	if count != 1 {
		t.Errorf("Expected 1, got %v", count)
	}

	// Case 4: Count After Deleting a Session
	sm.SessionDestroy("sessionid123")
	count = sm.SessionCount()
	if count != 1 {
		t.Errorf("Expected 1, got %v", count)
	}

	// Case 5: Concurrent Session Creation and Deletion
	smConcurrent := New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			smConcurrent.SessionCreate(fmt.Sprintf("sessionid%d", i))
		}(i)
	}
	wg.Wait()

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			smConcurrent.SessionDestroy(fmt.Sprintf("sessionid%d", i))
		}(i)
	}
	wg.Wait()

	count = smConcurrent.SessionCount()
	if count != 50 {
		t.Errorf("Expected 50, got %v", count)
	}
}

func TestSessionManager_SessionRefresh(t *testing.T) {
	sm := New()

	// Case 1: Refresh Existing Session
	sm.SessionCreate("sessionid123")
	s, err := sm.SessionRefresh("sessionid123", "sessionid456")
	if err != nil || s.sessionId != "sessionid456" {
		t.Errorf("Expected sessionid456, got %v, error: %v", s.sessionId, err)
	}

	// Case 2: Refresh Non-Existent Session
	s, err = sm.SessionRefresh("nonexistent", "sessionid789")
	if err != nil || s.sessionId != "sessionid789" {
		t.Errorf("Expected sessionid789, got %v, error: %v", s.sessionId, err)
	}

	// Case 3: Refresh with Same Session ID
	sm.SessionCreate("sessionid123")
	s, err = sm.SessionRefresh("sessionid123", "sessionid123")
	if err != nil || s.sessionId != "sessionid123" {
		t.Errorf("Expected sessionid123, got %v, error: %v", s.sessionId, err)
	}

	// Case 4: Concurrent Session Refresh
	smConcurrent := New()
	for i := 0; i < 100; i++ {
		smConcurrent.SessionCreate(fmt.Sprintf("sessionid%d", i))
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			smConcurrent.SessionRefresh(fmt.Sprintf("sessionid%d", i), fmt.Sprintf("newsessionid%d", i))
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		sid := fmt.Sprintf("newsessionid%d", i)
		if !smConcurrent.SessionExist(sid) {
			t.Errorf("Expected %v to exist", sid)
		}
	}
}

func TestSessionManager_SessionExist(t *testing.T) {
	sm := New()
	sm.SessionCreate("sessionid123")

	// Case 1: Session Exists
	exists := sm.SessionExist("sessionid123")
	if !exists {
		t.Errorf("Expected sessionid123 to exist")
	}

	// Case 2: Session Does Not Exist
	exists = sm.SessionExist("sessionid456")
	if exists {
		t.Errorf("Expected sessionid456 to not exist")
	}

	// Case 3: Session Deleted
	sm.SessionDestroy("sessionid123")
	exists = sm.SessionExist("sessionid123")
	if exists {
		t.Errorf("Expected sessionid123 to not exist after deletion")
	}

	// Case 4: Empty Session ID
	exists = sm.SessionExist("")
	if exists {
		t.Errorf("Expected empty session ID to not exist")
	}

	// Case 5: Concurrent Session Existence Checks
	smConcurrent := New()
	for i := 0; i < 100; i++ {
		smConcurrent.SessionCreate(fmt.Sprintf("sessionid%d", i))
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sid := fmt.Sprintf("sessionid%d", i)
			if !smConcurrent.SessionExist(sid) {
				t.Errorf("Expected %v to exist", sid)
			}
		}(i)
	}
	wg.Wait()
}

func TestSessionManager_SessionUpdate(t *testing.T) {
	sm := New()
	sm.SessionCreate("sessionid123")

	// Case 1: Update Existing Session
	err := sm.SessionUpdate("sessionid123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify that the session's lastAccessed time was updated
	session, _ := sm.sessions["sessionid123"]
	if time.Since(session.lastAccessed) > time.Second {
		t.Errorf("Expected lastAccessed to be updated recently, got %v", session.lastAccessed)
	}

	// Case 2: Update Non-Existent Session
	err = sm.SessionUpdate("sessionid456")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Case 3: Concurrent Session Updates
	smConcurrent := New()
	for i := 0; i < 100; i++ {
		smConcurrent.SessionCreate(fmt.Sprintf("sessionid%d", i))
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			smConcurrent.SessionUpdate(fmt.Sprintf("sessionid%d", i))
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		sid := fmt.Sprintf("sessionid%d", i)
		session, _ := smConcurrent.sessions[sid]
		if time.Since(session.lastAccessed) > time.Second {
			t.Errorf("Expected lastAccessed to be updated recently for %v, got %v", sid, session.lastAccessed)
		}
	}
}

func TestSessionManager_SessionDestroy(t *testing.T) {
	sm := New()
	sm.SessionCreate("sessionid123")

	// Case 1: Destroy Existing Session
	err := sm.SessionDestroy("sessionid123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Case 2: Destroy Non-Existent Session
	err = sm.SessionDestroy("sessionid456")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Case 3: Destroy Session Twice
	sm.SessionCreate("sessionid789")
	err = sm.SessionDestroy("sessionid789")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	err = sm.SessionDestroy("sessionid789")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Case 4: Destroy Session in Empty Session Manager
	smEmpty := New()
	err = smEmpty.SessionDestroy("sessionid123")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Case 5: Concurrent Session Destruction
	smConcurrent := New()
	for i := 0; i < 100; i++ {
		smConcurrent.SessionCreate(fmt.Sprintf("sessionid%d", i))
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			smConcurrent.SessionDestroy(fmt.Sprintf("sessionid%d", i))
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		sid := fmt.Sprintf("sessionid%d", i)
		if smConcurrent.SessionExist(sid) {
			t.Errorf("Expected %v to be destroyed", sid)
		}
	}
}

func TestSessionManager_SessionRead(t *testing.T) {
	sm := New()
	sm.SessionCreate("sessionid123")

	// Case 1: Read Existing Session from Cookie
	req := httptest.NewRequest("GET", "/", nil)
	cookie := &http.Cookie{Name: sm.Cookie.Name, Value: "sessionid123"}
	req.AddCookie(cookie)

	s, err := sm.SessionRead(req)
	if err != nil || s.sessionId != "sessionid123" {
		t.Errorf("Expected sessionid123, got %v, error: %v", s.sessionId, err)
	}

	// Case 2: Read Non-Existent Session
	req = httptest.NewRequest("GET", "/", nil)
	cookie = &http.Cookie{Name: sm.Cookie.Name, Value: "nonexistentsession"}
	req.AddCookie(cookie)

	s, err = sm.SessionRead(req)
	if err == nil || s != nil {
		t.Errorf("Expected error and nil session, got %v, error: %v", s, err)
	}

	// Case 3: Read Session with Invalid Cookie
	req = httptest.NewRequest("GET", "/", nil)
	cookie = &http.Cookie{Name: sm.Cookie.Name, Value: "%"}
	req.AddCookie(cookie)

	s, err = sm.SessionRead(req)
	if err == nil || s != nil {
		t.Errorf("Expected error and nil session, got %v, error: %v", s, err)
	}

	// Case 4: Read Session with No Cookie
	req = httptest.NewRequest("GET", "/", nil)

	s, err = sm.SessionRead(req)
	if err == nil || s != nil {
		t.Errorf("Expected error and nil session, got %v, error: %v", s, err)
	}

	// Case 5: Read Session from Header
	sm.Config.EnableHttpHeader = true
	sm.Config.SessionHeader = "Session-Id"
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set(sm.Config.SessionHeader, "sessionid123")

	s, err = sm.SessionRead(req)
	if err != nil || s.sessionId != "sessionid123" {
		t.Errorf("Expected sessionid123, got %v, error: %v", s.sessionId, err)
	}

	// Case 6: Concurrent Session Reads
	smConcurrent := New()
	for i := 0; i < 100; i++ {
		smConcurrent.SessionCreate(fmt.Sprintf("sessionid%d", i))
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/", nil)
			cookie := &http.Cookie{Name: smConcurrent.Cookie.Name, Value: fmt.Sprintf("sessionid%d", i)}
			req.AddCookie(cookie)
			s, err := smConcurrent.SessionRead(req)
			if err != nil || s.sessionId != fmt.Sprintf("sessionid%d", i) {
				t.Errorf("Expected sessionid%d, got %v, error: %v", i, s.sessionId, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestSessionManager_SessionCreate(t *testing.T) {
	sm := New()

	// Case 1: Create New Session
	s, err := sm.SessionCreate("sessionid123")
	if err != nil || s.sessionId != "sessionid123" {
		t.Errorf("Expected sessionid123, got %v, error: %v", s.sessionId, err)
	}

	// Case 2: Create Session with Existing ID
	s, err = sm.SessionCreate("sessionid123")
	if err != nil || s.sessionId != "sessionid123" {
		t.Errorf("Expected sessionid123, got %v, error: %v", s.sessionId, err)
	}

	// Case 3: Create Session with Empty ID
	_, err = sm.SessionCreate("")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Case 4: Concurrent Session Creation
	smConcurrent := New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sid := fmt.Sprintf("sessionid%d", i)
			s, err := smConcurrent.SessionCreate(sid)
			if err != nil || s.sessionId != sid {
				t.Errorf("Expected %v, got %v, error: %v", sid, s.sessionId, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestSessionManager_GlobalCleaner(t *testing.T) {
	// Case 1: Session Expires and Gets Cleaned
	sm := New()
	sm.Config.MaxLifetime = 1 * time.Second
	sm.SessionCreate("sessionid123")

	time.Sleep(2 * time.Second)
	sm.GlobalCleaner()

	if sm.SessionExist("sessionid123") {
		t.Errorf("Expected sessionid123 to be cleaned up")
	}

	// Case 2: Session Does Not Expire Before MaxLifetime
	sm = New()
	sm.Config.MaxLifetime = 3 * time.Second
	sm.SessionCreate("sessionid456")

	time.Sleep(1 * time.Second)
	sm.GlobalCleaner()

	if !sm.SessionExist("sessionid456") {
		t.Errorf("Expected sessionid456 to still exist")
	}

	// Case 3: No Sessions to Clean
	sm = New()
	sm.GlobalCleaner()
	// No sessions to check, just ensure no errors occur

	// Case 5: Concurrent Session Creation and Cleaning
	smConcurrent := New()
	smConcurrent.Config.MaxLifetime = 1 * time.Second
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			smConcurrent.SessionCreate(fmt.Sprintf("sessionid%d", i))
		}(i)
	}
	wg.Wait()

	time.Sleep(2 * time.Second)
	smConcurrent.GlobalCleaner()

	for i := 0; i < 100; i++ {
		sid := fmt.Sprintf("sessionid%d", i)
		if smConcurrent.SessionExist(sid) {
			t.Errorf("Expected %v to be cleaned up", sid)
		}
	}
}
