// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologicextension

import (
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
)

// Config has the configuration for the sumologic extension.
type Config struct {
	// squash ensures fields are correctly decoded in embedded struct.
	confighttp.HTTPClientSettings `mapstructure:",squash"`

	// Credentials contains Install Token for Sumo Logic service.
	// Please refer to https://help.sumologic.com/docs/manage/security/installation-tokens
	// for detailed instructions how to obtain the token.
	Credentials accessCredentials `mapstructure:",squash"`

	// CollectorName is the name under which collector will be registered.
	// Please note that registering a collector under a name which is already
	// used is not allowed.
	CollectorName string `mapstructure:"collector_name"`
	// CollectorEnvironment is the environment which will be used when updating
	// the collector metadata.
	CollectorEnvironment string `mapstructure:"collector_environment"`
	// CollectorDescription is the description which will be used when the
	// collector is being registered.
	CollectorDescription string `mapstructure:"collector_description"`
	// CollectorCategory is the collector category which will be used when the
	// collector is being registered.
	CollectorCategory string `mapstructure:"collector_category"`
	// CollectorFields defines the collector fields.
	// For more information on this subject visit:
	// https://help.sumologic.com/docs/manage/fields
	CollectorFields map[string]interface{} `mapstructure:"collector_fields"`

	// DiscoverCollectorTags enables collector metadata tag auto-discovery.
	DiscoverCollectorTags bool `mapstructure:"discover_collector_tags"`

	ApiBaseUrl string `mapstructure:"api_base_url"`

	HeartBeatInterval time.Duration `mapstructure:"heartbeat_interval"`

	// CollectorCredentialsDirectory is the directory where state files
	// with collector credentials will be stored after successful collector
	// registration. Default value is $HOME/.sumologic-otel-collector
	CollectorCredentialsDirectory string `mapstructure:"collector_credentials_directory"`

	// Clobber defines whether to delete any existing collector with the same
	// name and create a new one upon registration.
	// By default this is false.
	Clobber bool `mapstructure:"clobber"`

	// ForceRegistration defines whether to force registration every time the
	// collector starts.
	// This will cause the collector to not look at the locally stored credentials
	// and to always reach out to API to register itself.
	//
	// NOTE: if clobber is unset (default) then setting this to true will create
	// a new collector on Sumo UI on every collector start.
	//
	// By default this is false.
	ForceRegistration bool `mapstructure:"force_registration"`

	// Ephemeral defines whether the collector will be deleted after 12 hours
	// of inactivity.
	// By default this is false.
	Ephemeral bool `mapstructure:"ephemeral"`

	// TimeZone defines the time zone of the Collector.
	// For a list of possible values, refer to the "TZ" column in
	// https://en.wikipedia.org/wiki/List_of_tz_database_time_zones#List.
	TimeZone string `mapstructure:"time_zone"`

	// BackOff defines configuration of collector registration backoff algorithm
	// Exponential algorithm is being used.
	// Please see following link for details: https://github.com/cenkalti/backoff
	BackOff backOffConfig `mapstructure:"backoff"`
}

type accessCredentials struct {
	InstallToken configopaque.String `mapstructure:"install_token"`
}

// backOff configuration. See following link for details:
// https://pkg.go.dev/github.com/cenkalti/backoff/v4#ExponentialBackOff
type backOffConfig struct {
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	MaxInterval     time.Duration `mapstructure:"max_interval"`
	MaxElapsedTime  time.Duration `mapstructure:"max_elapsed_time"`
}
