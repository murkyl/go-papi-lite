package papilite

import (
	"fmt"
	"testing"
)

// TestListAllUsers gets all access zones and then lists all the users in each access zone
func TestListAllUsers(t *testing.T) {
	TestSetup(t)
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
		t.Log("Unable to get access zone list")
	}
	for _, zone := range zoneList {
		t.Log(fmt.Sprintf("\n==========\n%s\n==========", zone.Name))
		userList, err := conn.GetUserList(zone.Name)
		if err != nil {
			t.Log(fmt.Sprintf("Unable to get user list for zone: %s", zone.Name))
		}
		for _, user := range userList {
			t.Log(user.Name)
		}
	}
	conn.Disconnect()
}
