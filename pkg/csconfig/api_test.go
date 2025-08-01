package csconfig

import (
	"net/netip"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/crowdsecurity/go-cs-lib/cstest"
	"github.com/crowdsecurity/go-cs-lib/ptr"

	"github.com/crowdsecurity/crowdsec/pkg/types"
)

func TestLoadLocalApiClientCfg(t *testing.T) {
	tests := []struct {
		name        string
		input       *LocalApiClientCfg
		expected    *ApiCredentialsCfg
		expectedErr string
	}{
		{
			name: "basic valid configuration",
			input: &LocalApiClientCfg{
				CredentialsFilePath: "./testdata/lapi-secrets.yaml",
			},
			expected: &ApiCredentialsCfg{
				URL:      "http://localhost:8080/",
				Login:    "test",
				Password: "testpassword",
			},
		},
		{
			name: "invalid configuration",
			input: &LocalApiClientCfg{
				CredentialsFilePath: "./testdata/bad_lapi-secrets.yaml",
			},
			expected:    &ApiCredentialsCfg{},
			expectedErr: "field unknown_key not found in type csconfig.ApiCredentialsCfg",
		},
		{
			name: "invalid configuration filepath",
			input: &LocalApiClientCfg{
				CredentialsFilePath: "./testdata/nonexist_lapi-secrets.yaml",
			},
			expected:    nil,
			expectedErr: "open ./testdata/nonexist_lapi-secrets.yaml: " + cstest.FileNotFoundMessage,
		},
		{
			name: "valid configuration with insecure skip verify",
			input: &LocalApiClientCfg{
				CredentialsFilePath: "./testdata/lapi-secrets.yaml",
				InsecureSkipVerify:  ptr.Of(false),
			},
			expected: &ApiCredentialsCfg{
				URL:      "http://localhost:8080/",
				Login:    "test",
				Password: "testpassword",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Load()
			cstest.RequireErrorContains(t, err, tc.expectedErr)

			if tc.expectedErr != "" {
				return
			}

			assert.Equal(t, tc.expected, tc.input.Credentials)
		})
	}
}

func TestLoadOnlineApiClientCfg(t *testing.T) {
	tests := []struct {
		name        string
		input       *OnlineApiClientCfg
		expected    *ApiCredentialsCfg
		expectedErr string
	}{
		{
			name: "basic valid configuration",
			input: &OnlineApiClientCfg{
				CredentialsFilePath: "./testdata/online-api-secrets.yaml",
			},
			expected: &ApiCredentialsCfg{
				URL:      "http://crowdsec.api",
				Login:    "test",
				Password: "testpassword",
				PapiURL:  types.PAPIBaseURL,
			},
		},
		{
			name: "invalid configuration",
			input: &OnlineApiClientCfg{
				CredentialsFilePath: "./testdata/bad_lapi-secrets.yaml",
			},
			expected:    &ApiCredentialsCfg{},
			expectedErr: "failed to parse api server credentials",
		},
		{
			name: "missing field configuration",
			input: &OnlineApiClientCfg{
				CredentialsFilePath: "./testdata/bad_online-api-secrets.yaml",
			},
			expected: nil,
		},
		{
			name: "invalid configuration filepath",
			input: &OnlineApiClientCfg{
				CredentialsFilePath: "./testdata/nonexist_online-api-secrets.yaml",
			},
			expected:    &ApiCredentialsCfg{},
			expectedErr: "open ./testdata/nonexist_online-api-secrets.yaml: " + cstest.FileNotFoundMessage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Load()
			cstest.RequireErrorContains(t, err, tc.expectedErr)

			if tc.expectedErr != "" {
				return
			}

			assert.Equal(t, tc.expected, tc.input.Credentials)
		})
	}
}

func TestLoadAPIServer(t *testing.T) {
	tmpLAPI := &LocalApiServerCfg{
		ProfilesPath: "./testdata/profiles.yaml",
	}
	err := tmpLAPI.LoadProfiles()
	require.NoError(t, err)

	logLevel := log.InfoLevel
	config := &Config{}
	fcontent, err := os.ReadFile("./testdata/config.yaml")
	require.NoError(t, err)

	configData := os.ExpandEnv(string(fcontent))

	dec := yaml.NewDecoder(strings.NewReader(configData))
	dec.KnownFields(true)

	err = dec.Decode(&config)
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       *Config
		expected    *LocalApiServerCfg
		expectedErr string
	}{
		{
			name: "basic valid configuration",
			input: &Config{
				Self: []byte(configData),
				API: &APICfg{
					Server: &LocalApiServerCfg{
						ListenURI: "http://crowdsec.api",
						OnlineClient: &OnlineApiClientCfg{
							CredentialsFilePath: "./testdata/online-api-secrets.yaml",
						},
						ProfilesPath: "./testdata/profiles.yaml",
						PapiLogLevel: &logLevel,
					},
				},
				DbConfig: &DatabaseCfg{
					Type:   "sqlite",
					DbPath: "./testdata/test.db",
				},
				Common: &CommonCfg{
					LogDir:   "./testdata",
					LogMedia: "stdout",
				},
				DisableAPI: false,
			},
			expected: &LocalApiServerCfg{
				Enable:    ptr.Of(true),
				ListenURI: "http://crowdsec.api",
				TLS:       nil,
				DbConfig: &DatabaseCfg{
					DbPath:           "./testdata/test.db",
					Type:             "sqlite",
					MaxOpenConns:     DEFAULT_MAX_OPEN_CONNS,
					UseWal:           ptr.Of(true), // autodetected
					DecisionBulkSize: defaultDecisionBulkSize,
				},
				ConsoleConfigPath: DefaultConfigPath("console.yaml"),
				ConsoleConfig: &ConsoleConfig{
					ShareManualDecisions:  ptr.Of(false),
					ShareTaintedScenarios: ptr.Of(true),
					ShareCustomScenarios:  ptr.Of(true),
					ShareContext:          ptr.Of(false),
					ConsoleManagement:     ptr.Of(false),
				},
				LogDir:   "./testdata",
				LogMedia: "stdout",
				OnlineClient: &OnlineApiClientCfg{
					CredentialsFilePath: "./testdata/online-api-secrets.yaml",
					Credentials: &ApiCredentialsCfg{
						URL:      "http://crowdsec.api",
						Login:    "test",
						Password: "testpassword",
						PapiURL:  types.PAPIBaseURL,
					},
					Sharing: ptr.Of(true),
					PullConfig: CapiPullConfig{
						Community:  ptr.Of(true),
						Blocklists: ptr.Of(true),
					},
				},
				Profiles:               tmpLAPI.Profiles,
				ProfilesPath:           "./testdata/profiles.yaml",
				UseForwardedForHeaders: false,
				PapiLogLevel:           &logLevel,
				AutoRegister: &LocalAPIAutoRegisterCfg{
					Enable:              ptr.Of(false),
					Token:               "",
					AllowedRanges:       nil,
					AllowedRangesParsed: nil,
				},
			},
		},
		{
			name: "basic invalid configuration",
			input: &Config{
				Self: []byte(configData),
				API: &APICfg{
					Server: &LocalApiServerCfg{
						ListenURI: "http://crowdsec.api",
					},
				},
				Common: &CommonCfg{
					LogDir:   "./testdata/",
					LogMedia: "stdout",
				},
				DisableAPI: false,
			},
			expected: &LocalApiServerCfg{
				Enable:       ptr.Of(true),
				PapiLogLevel: &logLevel,
			},
			expectedErr: "no database configuration provided",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.LoadAPIServer(false, false)
			cstest.RequireErrorContains(t, err, tc.expectedErr)

			if tc.expectedErr != "" {
				return
			}

			assert.Equal(t, tc.expected, tc.input.API.Server)
		})
	}
}

func TestParseCapiWhitelists(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *CapiWhitelist
		expectedErr string
	}{
		{
			name:  "empty file",
			input: "",
			expected: &CapiWhitelist{
				Ips:   []netip.Addr{},
				Cidrs: []netip.Prefix{},
			},
			expectedErr: "empty file",
		},
		{
			name:  "empty ip and cidr",
			input: `{"ips": [], "cidrs": []}`,
			expected: &CapiWhitelist{
				Ips:   []netip.Addr{},
				Cidrs: []netip.Prefix{},
			},
		},
		{
			name:  "some ip",
			input: `{"ips": ["1.2.3.4"]}`,
			expected: &CapiWhitelist{
				Ips:   []netip.Addr{netip.MustParseAddr("1.2.3.4")},
				Cidrs: []netip.Prefix{},
			},
		},
		{
			name:  "some cidr",
			input: `{"cidrs": ["1.2.3.0/24"]}`,
			expected: &CapiWhitelist{
				Ips:   []netip.Addr{},
				Cidrs: []netip.Prefix{netip.MustParsePrefix("1.2.3.0/24")},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wl, err := parseCapiWhitelists(strings.NewReader(tc.input))
			cstest.RequireErrorContains(t, err, tc.expectedErr)

			if tc.expectedErr != "" {
				return
			}

			assert.Equal(t, tc.expected, wl)
		})
	}
}
