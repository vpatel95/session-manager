# Session Manager

## Table of Contents
+ [About](#about)
+ [Install](#install)
+ [Usage](#usage)
+ [Example](#example)

## About <a name = "about"></a>
Light-weight session in-memory session manager for golang

## Install <a name = "install"></a>

```
$ go get github.com/vpatel95/session-manager
```

## Usage <a name = "usage"></a>

1. Import the package using  
    ```go
    import sm "github.com/vpatel95/session-manager"
    ```

2. Create a SessionManager object 
   ```go
   manager := sm.New()

   // The default configs for session manager and cookie will be as follow
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
   ```
        
5. SessionManager Operations
    ```go
    func (sm *SessionManager) GetSessionId(r *http.Request) (string, error) 		// extract session ID from request
    func (sm *SessionManager) ListSessions() 				    		// Print all the sessions in the manager
    func (sm *SessionManager) SessionCount() int			    		// returns number of sessions in the manager
    func (sm *SessionManager) SessionRefresh(oldSid, sid string) (*Session, error)	// change the session Id for the session
    func (sm *SessionManager) SessionExist(sid string) bool				// check if session with session Id exists
    func (sm *SessionManager) SessionUpdate(sid string) error 				// update last access time for the session
    func (sm *SessionManager) SessionDestroy(sid string) error 				// delete session with given session Id
    func (sm *SessionManager) SessionRead(r *http.Request) (*Session, error) 		// retreive the session
    func (sm *SessionManager) SessionCreate(sid string) (*Session, error) 		// create a new session
    func (sm *SessionManager) SessionReadOrCreate(r *http.Request) (*Session, error)    // retreive the session, if not existing create a new session
    ```
    
6. Session operations
    ```
    func (s *Session) Get(key interface{}) interface{}	// to get value for 'key' from the session
    func (s *Session) Set(key, sd interface{}) error    // to set value for 'key in the session
    func (s *Session) Exist(key interface{}) bool      	// returns bool if 'key' exists in the session
    func (s *Session) Delete(key interface{}) error     // delete 'key' from the session
    ```
    
## Example <a name = "example"></a>

This is an example of a middleware that verifies the session and sets "user" key in the session

```go
func ValidateSessionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessId, err := sessManager.GetSessionId(r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		data, err := getTokenData(sessId)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		var user User
		user.Load(data["user"].(JSON))

		sess, err := sessManager.SessionReadOrCreate(r)
		if err != nil {
			log.Println("[ValidateSessionID] ::: Failed to get session : " + err.Error())
			next.ServeHTTP(w, r)
			return
		}

		sess.Set("user", user)
		next.ServeHTTP(w, r)
	})
}
```
