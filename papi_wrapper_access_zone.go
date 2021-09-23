package papilite

import (
	"github.com/mitchellh/mapstructure"
)

// GetAccessZoneList returns a list of all the access zones on a cluster
func (conn *OnefsConn) GetAccessZoneList() ([]OnefsAccessZone, error) {
	jsonObj, err := conn.Papi.Send(
		"GET",
		conn.PlatformPath+"/zones",
		nil, // query
		nil, // body
		nil, // extra headers
	)
	if err != nil {
		return nil, err
	}
	//conn.Logger().Debug(fmt.Sprintf("[GetAccessZoneList] JSON: %s", debug_json(jsonObj)))
	var result struct{ Zones []OnefsAccessZone }
	err = mapstructure.Decode(jsonObj, &result)
	if err != nil {
		return nil, err
	}
	return result.Zones, err
}
