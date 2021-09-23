# go-papi-lite

go-papi-lite is a lightweight wrapper for interacting with the PowerScale OneFS API. The API itself is often referred to as PAPI, or Platform API. The main goal of this library is to handle the session creation and tear down as well as automatically combine calls that would return pagination into a single request. The secondary goal is to have a minimal set of dependencies outside of the core Go libraries.
The library is split into 2 sections. The most basic part of the library handles the session and provides basic send commands. The second part of the library wraps the session and send command and provides functions that encapsulate parsing of the responses returned from the API.

## Basic code

The basic papi_lite.go provides a thin wrapper around native Go HTTP calls to handle PAPI session state. The wrapper also automatically makes multiple calls on behalf of the caller to combine any responses that have a resume token into a single response. If a session expires the module will attempt to automatically re-authenticate. If the API wrapper is used, then the basic calls do not normally need to be used directly. However, any call that is not present in the wrapper layer would have to use the underlying basic calls. A session context is required for calls and a function to return back the session context is provided by the NewSession function.

## Example

Get the PAPI version of the cluster

```go
conn := NewSession("")
conn.SetEndpoint("[http://fqdn.cluster.com:8080](http://fqdn.cluster.com:8080)")
conn.SetUser("api_user")
conn.SetPassword("user_password")
conn.SetIgnoreCert(true)
err := conn.Connect()
if err != nil {
	fmt.Printf("Error: %s\n", err)
}
jsonObj, err := conn.Send(
	"GET",
	"platform/latest",
	nil, // query args
	nil, // body
	nil, // extra headers
)
if err != nil {
	fmt.Printf("Error: %s\n", err)
}
fmt.Printf("JSON data: %v\n", jsonObj)
conn.Disconnect()
```

## Wrapper code

The wrapper code provides automatic parsing of responses from PAPI. The parsing of the JSON response relies on an external library. The data structures that contain the data are detailed in the papi_wrapper.go file.
There are a limited number of wrapper calls available and the calls are split into the main functional sections of the API.

## Example

Create a connection and list all users in the System zone

```go
conn := NewPapiConn()
conn.Connect(&OnefsCfg{
		User:       TestUser,
		Password:   TestPassword,
		Endpoint:   TestEndpoint,
		BypassCert: true,
	},
)
zoneList, err := conn.GetAccessZoneList()
if err != nil {
	fmt.Println("Unable to get access zone list")
}
for _, zone := range zoneList {
	fmt.Printf(fmt.Sprintf("\n==========\n%s\n==========\n", zone.Name))
	userList, err := conn.GetUserList(zone.Name)
	if err != nil {
		fmt.Printf("Unable to get user list for zone: %s\n", zone.Name)
	}
	for _, user := range userList {
		fmt.Println(user.Name)
	}
}
conn.Disconnect()
```
