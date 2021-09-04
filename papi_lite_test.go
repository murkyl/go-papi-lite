package papilite

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

var (
	TestEndpoint string
	TestUser     string
	TestPassword string
)

func envOrDefault(name string, defValue string) string {
	value, exist := os.LookupEnv(name)
	if exist == false {
		return defValue
	}
	return value
}

func envOrFail(t *testing.T, name string) string {
	value, exist := os.LookupEnv(name)
	if exist == false {
		t.Fatal("To run tests you must provide environment variables USER, PASSWORD, and ENDPOINT")
	}
	return value
}

func TestSetup(t *testing.T) {
	TestUser = envOrFail(t, "USER")
	TestPassword = envOrFail(t, "PASSWORD")
	TestEndpoint = envOrFail(t, "ENDPOINT")
}

// TestSessionTimeout requires the test to actually run for 905 seconds. The default session inactive timeout is 900 seconds.
// This may require the test to increase total timeout time as follows:
// go test -timeout 1000s
func TestSessionTimeout(t *testing.T) {
	if os.Getenv("TESTTIMEOUT") == "" {
		t.Skipf("Set environment variable TESTTIMEOUT to 1 to run the timeout test\nTestSessionTimeout requires the test to actually run for 905 seconds. The default session inactive timeout is 900 seconds. This may require the test to increase total timeout time as follows:\n    go test -timeout 1000s")
	}
	conn := NewSession("")
	conn.SetEndpoint(TestEndpoint)
	conn.SetUser(TestUser)
	conn.SetPassword(TestPassword)
	conn.SetIgnoreCert(true)
	err := conn.Connect()
	if err != nil {
		log.Print(fmt.Sprintf("Unable to connect to API endpoint: %s\n", err))
		return
	}
	log.Print(fmt.Sprintf("Connected to PAPI with session ID: %s", conn.SessionToken))
	_, err = GetPlatformLatest(conn)
	if err != nil {
		t.Errorf("Could not get the platform API version")
	}
	// Sleep for 900 + 5 seconds to exceed the session inactive timer
	time.Sleep(905 * time.Second)
	_, err = GetPlatformLatest(conn)
	if err != nil {
		t.Errorf("Did not re-authenticate properly")
	}
}

func GetPlatformLatest(conn *PapiSession) (string, error) {
	jsonObj, err := conn.Send(
		"GET",
		defaultPapiWrapperLatestPath,
		nil, // query args
		nil, // body
		nil, // extra headers
	)
	if err != nil {
		return "", err
	}
	return jsonObj["latest"].(string), nil
}
