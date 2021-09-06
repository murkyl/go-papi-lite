package papilite

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"log"
)

// CreateUser creates a new user in a given access zone
// This function only provides some basic user configuration options like home directory and primary group
func (conn *OnefsConn) CreateUser(name string, homedir string, pgroup string, zone string) (map[string]interface{}, error) {
	body := OnefsUser{
		PrimaryGroup: OnefsID{
			ID: "GROUP:" + pgroup,
		},
		Enabled:       true,
		Name:          name,
		HomeDirectory: homedir,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	if zone == "" {
		zone = "System"
	}
	jsonBody, err := conn.Papi.Send(
		"POST",
		conn.PlatformPath+"/auth/users",
		map[string]string{"force": "True", "zone": zone},
		bodyJSON, // body
		nil,      // extra headers
	)
	return jsonBody, err
}

// GetUserList returns a list of OnefsUsers in a given access zone
func (conn *OnefsConn) GetUserList(zone string) ([]OnefsUser, error) {
	jsonObj, err := conn.Papi.Send(
		"GET",
		conn.PlatformPath+"/auth/users",
		map[string]string{"zone": zone},
		nil, // body
		nil, // extra headers
	)
	if err != nil {
		return nil, err
	}
	//log.Print(fmt.Sprintf("[GetUserList] JSON: %s", debug_json(jsonObj)))
	var result struct{ Users []OnefsUser }
	err = mapstructure.Decode(jsonObj, &result)
	if err != nil {
		return nil, err
	}
	return result.Users, err
}

// GetUser returns the OnefsUser structure for a specific user
func (conn *OnefsConn) GetUser(name string, zone string) (*OnefsUser, error) {
	jsonObj, err := conn.Papi.Send(
		"GET",
		conn.PlatformPath+"/auth/users/"+name,
		map[string]string{"query_member_of": "True", "zone": zone},
		nil, // body
		nil, // extra headers
	)
	if err != nil {
		return nil, err
	}
	//log.Print(fmt.Sprintf("[GetUser] JSON: %s", debug_json(jsonObj)))
	var result struct{ Users []OnefsUser }
	err = mapstructure.Decode(jsonObj, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Users) < 1 {
		return nil, fmt.Errorf("[GetUser] User list was empty. Expected at least 1 user")
	}
	return &result.Users[0], err
}

// SetUserSuplementalGroups adds a list of groups to a user. This is done by repeated calls to AddUserToGroup
func (conn *OnefsConn) SetUserSuplementalGroups(name string, groups []string, zone string) error {
	errorCount := 0
	for i := 0; i < len(groups); i++ {
		_, err := conn.AddUserToGroup(name, groups[i], zone)
		if err != nil {
			log.Print(fmt.Sprintf("Unable to add user %s to group %s in access zone %s", name, groups[i], zone))
			errorCount++
		}
	}
	if errorCount > 0 {
		return fmt.Errorf("[SetUserSuplementalGroups] %d error(s) encountered adding user to groups: %s", errorCount, groups)
	}
	return nil
}

// AddUserToGroup will add a supplementary groups to a user
func (conn *OnefsConn) AddUserToGroup(name string, group string, zone string) (map[string]interface{}, error) {
	body := OnefsID{
		Name: name,
		Type: "user",
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	//log.Print(fmt.Sprintf("[AddUserToGroup] Body of request: %s", bodyJSON))
	jsonObj, err := conn.Papi.Send(
		"POST",
		conn.PlatformPath+"/auth/groups/"+group+"/members",
		map[string]string{"zone": zone},
		bodyJSON, // body
		nil,      // extra headers
	)
	if err != nil {
		// For this call, some errors can be safely ignored. Specifically if the user is already a member of one of the groups passed in there is no problem
		var apiErr struct{ Errors []OnefsError }
		apiDecodeErr := mapstructure.Decode(err, &apiErr)
		if apiDecodeErr != nil {
			log.Print(fmt.Sprintf("[AddUserToGroup] Request error: %s", err))
			return nil, err
		}
		duplicate := false
		for i := 0; i < len(apiErr.Errors); i++ {
			if apiErr.Errors[i].Code == "AEC_CONFLICT" {
				duplicate = true
			}
		}
		if !duplicate {
			return nil, err
		}
	}
	//log.Print(fmt.Sprintf("[AddUserToGroup] Response JSON: %s", debug_json(jsonObj)))
	return jsonObj, err
}

// DeleteUser will delete a user
func (conn *OnefsConn) DeleteUser(name string, zone string) (map[string]interface{}, error) {
	jsonObj, err := conn.Papi.Send(
		"DELETE",
		conn.PlatformPath+"/auth/users/"+name,
		map[string]string{"zone": zone},
		nil, // body
		nil, // extra headers
	)
	//log.Print(fmt.Sprintf("[DeleteUser] JSON: %s", debug_json(jsonObj)))
	if err != nil {
		log.Print(fmt.Sprintf("[DeleteUser] Error: %s", err))
		return nil, err
	}
	return jsonObj, err
}
