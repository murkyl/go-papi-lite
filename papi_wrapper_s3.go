package papilite

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
)

// GetS3Token creates a new S3 access secret. Returns a structure containing the current and former access keys and secrets.
// The call will always force a new key to be generated which will cause the old key to be invalidated after TTL minutes or immediately if no TTL is specified
// name: User name
// zone: Access zone for the request. Defaults to "System" if the string is empty
// ttl: Time in minutes to expire the old key. Defaults to no expiration if ttl is set to 0
func (conn *OnefsConn) GetS3Token(name string, zone string, ttl int) (*OnefsS3Key, error) {
	var bodyJSON []byte
	var err error
	if ttl > 0 {
		body := struct {
			TTL int `json:"existing_key_expiry_time"`
		}{TTL: ttl}
		bodyJSON, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	} else {
		bodyJSON = nil
	}
	if zone == "" {
		zone = "System"
	}
	//conn.Logger().Debug(fmt.Sprintf("[GetS3Token] S3 token body request: %s", bodyJSON))
	jsonObj, err := conn.Papi.Send(
		"POST",
		conn.PlatformPath+"/protocols/s3/keys/"+name,
		map[string]string{"force": "true", "zone": zone},
		bodyJSON, // body
		nil,      // extra headers
	)
	if err != nil {
		return nil, err
	}
	//conn.Logger().Debug(fmt.Sprintf("[GetS3Token] JSON: %s", debug_json(jsonObj)))
	var result struct{ Keys OnefsS3Key }
	err = mapstructure.Decode(jsonObj, &result)
	if err != nil {
		return nil, err
	}
	return &result.Keys, err
}
