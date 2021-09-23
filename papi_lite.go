// go-papi-lite is a lightweight wrapper for interacting with the PowerScale OneFS API. The API itself is often referred to as PAPI, or Platform API. The main goal of this library is to handle the session creation and tear down as well as automatically combine calls that would return pagination into a single request. The secondary goal is to have a minimal set of dependencies outside of the core Go libraries.
// The library is split into 2 sections. The most basic part of the library handles the session and provides basic send commands. The second part of the library wraps the session and send command and provides functions that encapsulate parsing of the responses returned from the API.
//
// Basic code
//
// The basic papi_lite.go provides a thin wrapper around native Go HTTP calls to handle PAPI session state. The wrapper also automatically makes multiple calls on behalf of the caller to combine any responses that have a resume token into a single response. If a session expires the module will attempt to automatically re-authenticate. If the API wrapper is used, then the basic calls do not normally need to be used directly. However, any call that is not present in the wrapper layer would have to use the underlying basic calls. A session context is required for calls and a function to return back the session context is provided by the NewSession function.
//
// Example
//
// Get the PAPI version of the cluster
//
// 	conn := NewSession("")
// 	conn.SetEndpoint("http://fqdn.cluster.com:8080")
// 	conn.SetUser("api_user")
// 	conn.SetPassword("user_password")
// 	conn.SetIgnoreCert(true)
// 	err := conn.Connect()
// 	if err != nil {
// 		fmt.Printf("Error: %s\n", err)
// 	}
// 	jsonObj, err := conn.Send(
// 		"GET",
// 		"platform/latest",
// 		nil, // query args
// 		nil, // body
// 		nil, // extra headers
// 	)
// 	if err != nil {
// 		fmt.Printf("Error: %s\n", err)
// 	}
// 	fmt.Printf("JSON data: %v\n", jsonObj)
// 	conn.Disconnect()
//
// Wrapper code
//
// The wrapper code provides automatic parsing of responses from PAPI. The parsing of the JSON response relies on an external library. The data structures that contain the data are detailed in the papi_wrapper.go file.
// There are a limited number of wrapper calls available and the calls are split into the main functional sections of the API.
//
// Example
//
// Create a connection and list all users in the System zone
//
// 	conn := NewPapiConn()
// 	conn.Connect(&OnefsCfg{
// 			User:       TestUser,
// 			Password:   TestPassword,
// 			Endpoint:   TestEndpoint,
// 			BypassCert: true,
// 		},
// 	)
// 	zoneList, err := conn.GetAccessZoneList()
// 	if err != nil {
// 		fmt.Println("Unable to get access zone list")
// 	}
// 	for _, zone := range zoneList {
// 		fmt.Printf(fmt.Sprintf("\n==========\n%s\n==========\n", zone.Name))
// 		userList, err := conn.GetUserList(zone.Name)
//		if err != nil {
// 			fmt.Printf("Unable to get user list for zone: %s\n", zone.Name)
//		}
// 		for _, user := range userList {
// 			fmt.Println(user.Name)
//		}
// 	}
// 	conn.Disconnect()
//
package papilite

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultConnTimeout    int    = 120
	defaultMaxReauthCount int    = 1
	sessionPath           string = "session/1/session"
	maxCount              int    = 10000
)

// PapiSession represents the state object for a connection
type PapiSession struct {
	User         string
	Password     string
	Endpoint     string
	IgnoreCert   bool
	SessionToken string
	CsrfToken    string
	Client       *http.Client
	ConnTimeout  int
	reauthCount  int
}

// sessionRequest defines the parameters required in an HTTP POST body to create a session
// struct tags are used to make the field names lowercase as the Go default is to not marshall
// any struct members that do not start with an upper case letter
type sessionRequest struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Services []string `json:"services"`
}

// NewSession is a factory function returning a context object. This must be used in order to
// use any  of the other functions. This context can be modified by helper functions before
// connecting to the endpoint
func NewSession(endpoint string) *PapiSession {
	return &PapiSession{
		Endpoint:    endpoint,
		ConnTimeout: defaultConnTimeout,
		IgnoreCert:  false,
	}
}

// SetUser is a setter used to set the user name in the session context
func (ctx *PapiSession) SetUser(s string) string {
	old := ctx.User
	ctx.User = s
	return old
}

// SetPassword is a setter used to set the password in the session context
func (ctx *PapiSession) SetPassword(s string) string {
	old := ctx.Password
	ctx.Password = s
	return old
}

// SetEndpoint updates the endpoint that will be used for the PAPI connection
// The string passed in must include the protocol (http or https), end point, and port
// e.g. https://cluster.fqdn:8080
// If SetEndpoint is used after a connection has already been made you must disconnect
// and reconnect to use the new endpoint
func (ctx *PapiSession) SetEndpoint(s string) string {
	old := ctx.Endpoint
	ctx.Endpoint = s
	return old
}

// SetIgnoreCert is a setter used to set the flag to ignore or not ignore certificate checking
func (ctx *PapiSession) SetIgnoreCert(b bool) bool {
	old := ctx.IgnoreCert
	ctx.IgnoreCert = b
	return old
}

// SetConnTimeout is a setter used to set the timeout for the HTTP connection (http.Client)
func (ctx *PapiSession) SetConnTimeout(t int) int {
	old := ctx.ConnTimeout
	ctx.ConnTimeout = t
	return old
}

// GetURL takes in a path and query argument to create a full URL based on the Endpoint
// in the PapiSession.
// path can be a string or a slice/array of strings
// query is map of strings in a basic key, value pair
func (ctx *PapiSession) GetURL(path interface{}, query map[string]string) string {
	x, _ := url.Parse(ctx.Endpoint)
	switch path.(type) {
	case []string:
		x.Path += strings.Join(path.([]string), "/")
	default:
		x.Path += path.(string)
	}
	q := url.Values{}
	for k, v := range query {
		q.Add(k, v)
	}
	x.RawQuery = q.Encode()
	return x.String()
}

// init is an internal helper function to create the http.Client object
func (ctx *PapiSession) init() error {
	if ctx.IgnoreCert {
		ctx.Client = &http.Client{
			Timeout: time.Duration(ctx.ConnTimeout) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	} else {
		ctx.Client = &http.Client{
			Timeout: time.Duration(ctx.ConnTimeout) * time.Second,
		}
	}
	return nil
}

// Connect is called to initiate a connection to the endpoint. Connect can be called multiple times as
// the fucntion will automatically disconnect any existing connection. Changes to the endpoint can be
// made to the context and another Connect made to switch to the other endpoint.
func (ctx *PapiSession) Connect() error {
	var match []string
	// Regular expressions to pull the isisessid and isicsrf fields out of the Cookie header in the session response
	rexSession := regexp.MustCompile(`.*isisessid=(?P<session>[^;]+).*`)
	rexCsrf := regexp.MustCompile(`.*isicsrf=(?P<csrf>[^;]+).*`)

	// Cleanup any existing session before trying to connect
	ctx.Disconnect()
	// Automatically initialize the PapiSession if it is not already initialized
	if ctx.Client == nil {
		ctx.init()
	}

	body := sessionRequest{
		Username: ctx.User,
		Password: ctx.Password,
		Services: []string{"platform", "namespace"},
	}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", ctx.GetURL(sessionPath, nil), bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("[Connect] Failed to create NewRequest: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	resp, err := ctx.Client.Do(req)
	if err != nil {
		return fmt.Errorf("[Connect] Client.Do error: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("[Connect] Unable to create a session: %s", fmt.Sprintf("%+v", string(respBody)))
	}
	sessionID := resp.Header["Set-Cookie"]
	for i := 0; i < len(sessionID); i++ {
		match = rexSession.FindStringSubmatch(sessionID[i])
		if match != nil {
			ctx.SessionToken = match[1]
			continue
		}
		match = rexCsrf.FindStringSubmatch(sessionID[i])
		if match != nil {
			ctx.CsrfToken = match[1]
			continue
		}
	}
	if ctx.SessionToken == "" {
		return errors.New("[Connect] No session token found in API connect call")
	}
	if ctx.CsrfToken == "" {
		return errors.New("[Connect] No CSRF token found in API connect call")
	}
	ctx.reauthCount = 0
	return nil
}

// Disconnect cleans up a connection to an endpoint. This should be called after calls to the API are completed
func (ctx *PapiSession) Disconnect() error {
	if ctx.Client == nil {
		return nil
	}
	req, err := http.NewRequest("DELETE", ctx.GetURL(sessionPath, nil), nil)
	if err != nil {
		return fmt.Errorf("[Disconnect] Failed to crate NewRequest: %v", err)
	}
	setHeaders(req, ctx, nil)
	_, err = ctx.Client.Do(req)
	if err != nil {
		err = fmt.Errorf("[Disconnect] Session delete error: %v", err)
	}
	ctx.Client.CloseIdleConnections()
	ctx.Client = nil
	ctx.SessionToken = ""
	ctx.CsrfToken = ""
	// This return takes the error code from the Client.Do above and returns it. Successful runs will return nil
	return err
}

// Reconnect is a simple helper function that calls Disconnect and then Connect in succession
func (ctx *PapiSession) Reconnect() error {
	ctx.Disconnect()
	return ctx.Connect()
}

// SendRaw makes a call to the API and returns the raw HTTP response and error codes. It is the responsibility
// of the caller to process the response.
func (ctx *PapiSession) SendRaw(method string, path interface{}, query map[string]string, body interface{}, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	switch body.(type) {
	case nil:
		reqBody = nil
	case []byte:
		reqBody = bytes.NewReader(body.([]byte))
	case string:
		reqBody = bytes.NewReader([]byte(body.(string)))
	default:
		reqBody = bytes.NewReader([]byte(body.(string)))
	}
	req, err := http.NewRequest(method, ctx.GetURL(path, query), reqBody)
	if err != nil {
		return nil, fmt.Errorf("[SendRaw] Request error: %v", err)
	}
	setHeaders(req, ctx, headers)
	return ctx.Client.Do(req)
}

// Send performs an API call and does some automatic post-processing. This processing consists of converting the
// response into a JSON object in the form of a map[string]interface{}. Any resume keys are automatically handled
// and the result is combined such that all values are returned in a single object. This may be a problem for very
// large data sets. In those situations use SendRaw as an alternative.
func (ctx *PapiSession) Send(method string, path interface{}, query map[string]string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	jsonBody := make(map[string]interface{})
	var jsonTemp map[string]interface{}
	var resumeKey string
	var rkey interface{}

	// The count variable puts an upper limit on the number of times this function will automatically fetch additional data
	for resume, count := true, 0; resume && count < maxCount; count++ {
		if resumeKey != "" {
			// When a resume key is used all old query parameters should be discarded and only the resume key in the query arguments list
			query = map[string]string{"resume": resumeKey}
		}
		resp, err := ctx.SendRaw(method, path, query, body, headers)
		if err != nil {
			return nil, fmt.Errorf("[Send] Error returned by SendRaw: %v", err)
		}
		defer resp.Body.Close()
		rawBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("[Send] Error reading response body: %v", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			if resp.StatusCode == 401 {
				// If a 401 error with a message of "Authorization required" is received, we should automatically re-authenticate to get a new session token and retry the request
				if ctx.reauthCount >= defaultMaxReauthCount {
					log.Printf("[ERROR][Send] Automatic re-authentication failed!")
				} else {
					ctx.reauthCount++
					ctx.Reconnect()
					// Recursively call Send with the same parameters and return the result. There is a limited number of re-auth attempts before failing the entire call
					return ctx.Send(method, path, query, body, headers)
				}
			}
			return nil, fmt.Errorf("[Send] Non 2xx response received (%d): %s", resp.StatusCode, fmt.Sprintf("%+v", string(rawBody)))
		}

		// If there is no body in the response, there is no need to try and process continuation requests
		// This can happen for some methods like DELETE
		if len(rawBody) == 0 || rawBody == nil {
			return nil, nil
		}

		err = json.Unmarshal(rawBody, &jsonTemp)
		if err != nil {
			return nil, fmt.Errorf("[Send] Error unmarshaling JSON: %v", err)
		}
		rkey, resume = jsonTemp["resume"]
		if resume == true {
			if rkey != nil {
				resumeKey = rkey.(string)
			} else {
				resume = false
			}
		}
		ekey, ok := jsonBody["errors"]
		if ok == true {
			return nil, fmt.Errorf("[Send] Response to Send request returned errors in JSON: %v", ekey)
		}
		// Remove extraneous fields from the JSON response as they are only used with continued responses
		delete(jsonTemp, "errors")
		delete(jsonTemp, "resume")
		delete(jsonTemp, "total")
		// Combine the jsonTemp with jsonBody
		for key, dval := range jsonTemp {
			sval, ok := jsonBody[key]
			if ok == true {
				switch sval.(type) {
				case []interface{}:
					// TODO: Use more efficient way to combine results
					for _, v := range dval.([]interface{}) {
						jsonBody[key] = append(jsonBody[key].([]interface{}), v)
					}
				default:
					jsonBody[key] = dval
				}
			} else {
				jsonBody[key] = dval
			}
		}
	}
	return jsonBody, nil
}

// setHeaders sets the headers for a request appropriately
// The function takes the request, PapiSession, and a map containing possible header key/value pairs
// The function first overwrites any existing headers in the request with those supplied in the headers parameter
// Only after this is done do we attempt to add in the session, CSRF and Referer headers. If these headers exist
// in the passed in headers array, they are not overriden. The values in the passed in headers map take precedence
func setHeaders(req *http.Request, ctx *PapiSession, headers map[string]string) {
	for k, v := range headers {
		// Manually set headers as we want to preserve the case sensitivity of each header
		req.Header[k] = []string{v}
	}
	defaultHeaders := map[string]string{
		"Accept":       "application/json",
		"Cookie":       "isisessid=" + ctx.SessionToken,
		"Content-Type": "application/json",
		"Referer":      ctx.Endpoint,
		"X-CSRF-Token": ctx.CsrfToken,
	}
	for k, v := range defaultHeaders {
		if _, ok := req.Header[k]; !ok {
			req.Header.Add(k, v)
		}
	}
}
