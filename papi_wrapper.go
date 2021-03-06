package papilite

import (
	"fmt"
	"log"
)

const (
	defaultPapiWrapperLatestPath   string = "platform/latest"
	defaultPapiWrapperPlatformPath string = "platform/10"
	defaultPapiWrapperRanPath      string = ""
	defaultPapiWrapperServicePath  string = ""
)

// OnefsCfg contains the configuration to connect to a OneFS cluster endpoint
type OnefsCfg struct {
	User       string
	Password   string
	Endpoint   string
	BypassCert bool
}

// OnefsConn contains the state of a connection
type OnefsConn struct {
	Papi         *PapiSession
	PlatformPath string
	RanPath      string
	ServicePath  string
}

// OnefsError is the structure of API call errors
type OnefsError struct {
	Code    string `json:"code,omitempty" mapstructure:"code,omitempty"`
	Message string `json:"message,omitempty" mapstructure:"message,omitempty"`
}

// OnefsID represents a generic persona object in the API
type OnefsID struct {
	ID   string `json:"id,omitempty" mapstructure:"id,omitempty"`
	Name string `json:"name,omitempty" mapstructure:"name,omitempty"`
	Type string `json:"type,omitempty" mapstructure:"type,omitempty"`
}

// OnefsS3Key represents the data values returned in an S3 key create call
type OnefsS3Key struct {
	AccessID           string `json:"access_id,omitempty" mapstructure:"access_id"`
	OldKeyExpiry       int    `json:"old_key_expiry,omitempty" mapstructure:"old_key_expiry"`
	OldKeyTimestamp    int    `json:"old_key_timestamp,omitempty" mapstructure:"old_key_timestamp"`
	SecretKey          string `json:"secret_key,omitempty" mapstructure:"secret_key"`
	SecretKeyTimestamp int    `json:"secret_key_timestamp,omitempty" mapstructure:"secret_key_timestamp"`
}

// OnefsUser represents a local user
type OnefsUser struct {
	Name          string    `json:"name" mapstructure:"name"`
	Email         string    `json:"email,omitempty" mapstructure:"email,omitempty"`
	Enabled       bool      `json:"enabled,omitempty" mapstructure:"enabled,omitempty"`
	Expiry        int       `json:"expiry,omitempty" mapstructure:"expiry,omitempty"`
	HomeDirectory string    `json:"home_directory,omitempty" mapstructure:"home_directory,omitempty"`
	MemberOf      []OnefsID `json:"member_of,omitempty" mapstructure:"member_of,omitempty"`
	PrimaryGroup  OnefsID   `json:"primary_group,omitempty" mapstructure:"primary_group,omitempty"`
	Shell         string    `json:"shell,omitempty" mapstructure:"shell,omitempty"`
}

// OnefsAccessZone represents an access zone
type OnefsAccessZone struct {
	AlternateSystemProvider  string    `json:"alternate_system_provider" mapstructure:"alternate_system_provider"`
	AuthProviders            []string  `json:"auth_providers" mapstructure:"auth_providers"`
	CacheEntryExpiry         int       `json:"cache_entry_expiry" mapstructure:"cache_entry_expirty"`
	Groupnet                 string    `json:"groupnet" mapstructure:"groupnet"`
	HomeDirectoryUmast       int       `json:"home_directory_umask" mapstructure:"home_directory_umask"`
	ID                       string    `json:"id" mapstructure:"id"`
	IfsRestricted            []OnefsID `json:"ifs_restricted" mapstructure:"ifs_restricted"`
	MapUntrusted             string    `json:"map_untrusted" mapstructure:"map_untrusted"`
	Name                     string    `json:"name" mapstructure:"name"`
	NegativeCacheEntryExpiry int       `json:"negative_cache_entry_expiry" mapstructure:"negative_cache_entry_expiry"`
	NetbiosName              string    `json:"netbios_name" mapstructure:"netbios_name"`
	Path                     string    `json:"path" mapstructure:"path"`
	SkeletonDirectory        string    `json:"skeleton_directory" mapstructure:"skeleton_directory"`
	System                   bool      `json:"system" mapstructure:"system"`
	SystemProvider           string    `json:"system_provider" mapstructure:"system_provider"`
	UserMappingRules         []string  `json:"user_mapping_rules" mapstructure:"user_mapping_rule"`
	ZoneID                   int       `json:"zone_id" mapstructure:"zone_id"`
}

// NewPapiConn returns a connection state object that is used by all other calls in this library
func NewPapiConn() *OnefsConn {
	return &OnefsConn{
		Papi:         NewSession(""),
		PlatformPath: defaultPapiWrapperPlatformPath,
		RanPath:      defaultPapiWrapperRanPath,
		ServicePath:  defaultPapiWrapperServicePath,
	}
}

// Connect performs the actual connection to the OneFS cluster endpoint given the endpoint configuration in a OnefsCfg struct
func (conn *OnefsConn) Connect(cfg *OnefsCfg) error {
	conn.Papi.Disconnect()
	conn.Papi.SetEndpoint(cfg.Endpoint)
	conn.Papi.SetUser(cfg.User)
	conn.Papi.SetPassword(cfg.Password)
	conn.Papi.SetIgnoreCert(cfg.BypassCert)
	err := conn.Papi.Connect()
	if err != nil {
		log.Print(fmt.Sprintf("[Connect] Unable to connect to API endpoint: %s\n", err))
		return err
	}
	//log.Print(fmt.Sprintf("[Connect] Connected to PAPI with session ID: %s", conn.Papi.SessionToken))
	apiVer, err := conn.GetPlatformLatest()
	if err != nil {
		log.Print("Unable to get latest platform API version automatically")
	} else {
		conn.PlatformPath = "platform/" + apiVer
	}
	return nil
}

// Disconnect disconnects the connection to the endpoint. This is safe to call multiple times and even if a connect was never performed
func (conn *OnefsConn) Disconnect() error {
	if conn.Papi != nil {
		err := conn.Papi.Disconnect()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPlatformLatest returns the current API version in string format of the connected OneFS cluster
func (conn *OnefsConn) GetPlatformLatest() (string, error) {
	jsonObj, err := conn.Papi.Send(
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
