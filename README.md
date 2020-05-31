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
go get github.com/vpatel95/session-manager
```

End with an example of getting some data out of the system or using it for a little demo.

## Usage <a name = "usage"></a>

1. Import the package using  
    ```
    import sess "github.com/vpatel95/session-manager"
    ```

2. Set the attributes in `session.json` file and put it in `<project_root>/configs/session.json`. An Example is given below  
    ```
    {  
        "cookieName": "sessionid",  
        "gcLifetime": 60,  
        "maxLifetime": 172800,  
        "HTTPOnly": true,  
        "secure": false,  
        "cookieLifeTime": 172800,  
        "domain": "",  
        "enableHTTPHeader": false,  
        "sessHeader": ""  
    }  
    ```
3. Create a SessionManager object or use default  
    - Default Session Manager can be accessed by
        ```
        sess.SessionManager
        ```
    - Create a new SessionManager
        ```
        manager := sess.NewSessionManager(sess.GetManagerConfig())
        ```
        
4. SessionManager Operations
    ```
    session, err := manager.SessionReadOrCreate(request)        // retreive the session, if not existing create a new session
    sessId, err := manager.GetSessionId(request)                // extract session ID from request
    ok := manager.SessionExist(sessId)                          // check if session with session Id exists
    session, err := manager.SessionRead(request)                // retreive the session
    err := manager.SessionUpdate(sessId)                        // update last access time for the session
    err := manager.SessionDestroy(sessId)                       // delete session with given session Id
    session, err := manager.SessionRefresh(oldSessId, sessId)   // change the session Id for the session
    count := manager.SessionCount()                             // returns number of sessions in the manager
    ```
5. Session operations
    ```
    session.Get(key)        // to get value for 'key' from the session
    session.Set(key)        // to set value for 'key in the session
    session.Exist(key)      // returns bool if 'key' exists in the session
    session.Delete(key)     // delete 'key' from the session
    ```
    
## Example <a name = "example"></a>

This is an example of a middleware that verifies the session and sets "user" key in the session

```
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
