/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2017 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package cloudapi

import (
	"encoding/json"
	"time"

	"gopkg.in/guregu/null.v3"

	"github.com/kelseyhightower/envconfig"
	"go.k6.io/k6/lib/types"
)

// Config holds all the necessary data and options for sending metrics to the Load Impact cloud.
//nolint: lll
type Config struct {
	PushRefID                       null.String        `json:"pushRefID" envconfig:"K6_CLOUD_PUSH_REF_ID"`
	Token                           null.String        `json:"token" envconfig:"K6_CLOUD_TOKEN"`
	Name                            null.String        `json:"name" envconfig:"K6_CLOUD_NAME"`
	Host                            null.String        `json:"host" envconfig:"K6_CLOUD_HOST"`
	WebAppURL                       null.String        `json:"webAppURL" envconfig:"K6_CLOUD_WEB_APP_URL"`
	LogsTailURL                     null.String        `json:"-" envconfig:"K6_CLOUD_LOGS_TAIL_URL"`
	MetricPushInterval              types.NullDuration `json:"metricPushInterval" envconfig:"K6_CLOUD_METRIC_PUSH_INTERVAL"`
	Timeout                         types.NullDuration `json:"timeout" envconfig:"K6_CLOUD_TIMEOUT"`
	AggregationOutlierIqrCoefLower  null.Float         `json:"aggregationOutlierIqrCoefLower" envconfig:"K6_CLOUD_AGGREGATION_OUTLIER_IQR_COEF_LOWER"`
	AggregationOutlierIqrRadius     null.Float         `json:"aggregationOutlierIqrRadius" envconfig:"K6_CLOUD_AGGREGATION_OUTLIER_IQR_RADIUS"`
	MaxMetricSamplesPerPackage      null.Int           `json:"maxMetricSamplesPerPackage" envconfig:"K6_CLOUD_MAX_METRIC_SAMPLES_PER_PACKAGE"`
	ProjectID                       null.Int           `json:"projectID" envconfig:"K6_CLOUD_PROJECT_ID"`
	MetricPushConcurrency           null.Int           `json:"metricPushConcurrency" envconfig:"K6_CLOUD_METRIC_PUSH_CONCURRENCY"`
	AggregationPeriod               types.NullDuration `json:"aggregationPeriod" envconfig:"K6_CLOUD_AGGREGATION_PERIOD"`
	AggregationCalcInterval         types.NullDuration `json:"aggregationCalcInterval" envconfig:"K6_CLOUD_AGGREGATION_CALC_INTERVAL"`
	AggregationWaitPeriod           types.NullDuration `json:"aggregationWaitPeriod" envconfig:"K6_CLOUD_AGGREGATION_WAIT_PERIOD"`
	AggregationMinSamples           null.Int           `json:"aggregationMinSamples" envconfig:"K6_CLOUD_AGGREGATION_MIN_SAMPLES"`
	AggregationOutlierAlgoThreshold null.Int           `json:"aggregationOutlierAlgoThreshold" envconfig:"K6_CLOUD_AGGREGATION_OUTLIER_ALGO_THRESHOLD"`
	AggregationOutlierIqrCoefUpper  null.Float         `json:"aggregationOutlierIqrCoefUpper" envconfig:"K6_CLOUD_AGGREGATION_OUTLIER_IQR_COEF_UPPER"`
	AggregationSkipOutlierDetection null.Bool          `json:"aggregationSkipOutlierDetection" envconfig:"K6_CLOUD_AGGREGATION_SKIP_OUTLIER_DETECTION"`
	StopOnError                     null.Bool          `json:"stopOnError" envconfig:"K6_CLOUD_STOP_ON_ERROR"`
	NoCompress                      null.Bool          `json:"noCompress" envconfig:"K6_CLOUD_NO_COMPRESS"`
}

// NewConfig creates a new Config instance with default values for some fields.
func NewConfig() Config {
	return Config{
		Host:                       null.NewString("https://ingest.k6.io", false),
		LogsTailURL:                null.NewString("wss://cloudlogs.k6.io/api/v1/tail", false),
		WebAppURL:                  null.NewString("https://app.k6.io", false),
		MetricPushInterval:         types.NewNullDuration(1*time.Second, false),
		MetricPushConcurrency:      null.NewInt(1, false),
		MaxMetricSamplesPerPackage: null.NewInt(100000, false),
		Timeout:                    types.NewNullDuration(1*time.Minute, false),
		// Aggregation is disabled by default, since AggregationPeriod has no default value
		// but if it's enabled manually or from the cloud service, those are the default values it will use:
		AggregationCalcInterval:         types.NewNullDuration(3*time.Second, false),
		AggregationWaitPeriod:           types.NewNullDuration(5*time.Second, false),
		AggregationMinSamples:           null.NewInt(25, false),
		AggregationOutlierAlgoThreshold: null.NewInt(75, false),
		AggregationOutlierIqrRadius:     null.NewFloat(0.25, false),

		// Since we're measuring durations, the upper coefficient is slightly
		// lower, since outliers from that side are more interesting than ones
		// close to zero.
		AggregationOutlierIqrCoefLower: null.NewFloat(1.5, false),
		AggregationOutlierIqrCoefUpper: null.NewFloat(1.3, false),
	}
}

// Apply saves config non-zero config values from the passed config in the receiver.
func (c Config) Apply(cfg Config) Config {
	if cfg.Token.Valid {
		c.Token = cfg.Token
	}
	if cfg.ProjectID.Valid && cfg.ProjectID.Int64 > 0 {
		c.ProjectID = cfg.ProjectID
	}
	if cfg.Name.Valid && cfg.Name.String != "" {
		c.Name = cfg.Name
	}
	if cfg.Host.Valid && cfg.Host.String != "" {
		c.Host = cfg.Host
	}
	if cfg.LogsTailURL.Valid && cfg.LogsTailURL.String != "" {
		c.LogsTailURL = cfg.LogsTailURL
	}
	if cfg.PushRefID.Valid {
		c.PushRefID = cfg.PushRefID
	}
	if cfg.WebAppURL.Valid {
		c.WebAppURL = cfg.WebAppURL
	}
	if cfg.NoCompress.Valid {
		c.NoCompress = cfg.NoCompress
	}
	if cfg.StopOnError.Valid {
		c.StopOnError = cfg.StopOnError
	}
	if cfg.Timeout.Valid {
		c.Timeout = cfg.Timeout
	}
	if cfg.MaxMetricSamplesPerPackage.Valid {
		c.MaxMetricSamplesPerPackage = cfg.MaxMetricSamplesPerPackage
	}
	if cfg.MetricPushInterval.Valid {
		c.MetricPushInterval = cfg.MetricPushInterval
	}
	if cfg.MetricPushConcurrency.Valid {
		c.MetricPushConcurrency = cfg.MetricPushConcurrency
	}

	if cfg.AggregationPeriod.Valid {
		c.AggregationPeriod = cfg.AggregationPeriod
	}
	if cfg.AggregationCalcInterval.Valid {
		c.AggregationCalcInterval = cfg.AggregationCalcInterval
	}
	if cfg.AggregationWaitPeriod.Valid {
		c.AggregationWaitPeriod = cfg.AggregationWaitPeriod
	}
	if cfg.AggregationMinSamples.Valid {
		c.AggregationMinSamples = cfg.AggregationMinSamples
	}
	if cfg.AggregationSkipOutlierDetection.Valid {
		c.AggregationSkipOutlierDetection = cfg.AggregationSkipOutlierDetection
	}
	if cfg.AggregationOutlierAlgoThreshold.Valid {
		c.AggregationOutlierAlgoThreshold = cfg.AggregationOutlierAlgoThreshold
	}
	if cfg.AggregationOutlierIqrRadius.Valid {
		c.AggregationOutlierIqrRadius = cfg.AggregationOutlierIqrRadius
	}
	if cfg.AggregationOutlierIqrCoefLower.Valid {
		c.AggregationOutlierIqrCoefLower = cfg.AggregationOutlierIqrCoefLower
	}
	if cfg.AggregationOutlierIqrCoefUpper.Valid {
		c.AggregationOutlierIqrCoefUpper = cfg.AggregationOutlierIqrCoefUpper
	}
	return c
}

// MergeFromExternal merges three fields from the JSON in a loadimpact key of
// the provided external map. Used for options.ext.loadimpact settings.
func MergeFromExternal(external map[string]json.RawMessage, conf *Config) error {
	if val, ok := external["loadimpact"]; ok {
		// TODO: Important! Separate configs and fix the whole 2 configs mess!
		tmpConfig := Config{}
		if err := json.Unmarshal(val, &tmpConfig); err != nil {
			return err
		}
		// Only take out the ProjectID, Name and Token from the options.ext.loadimpact map:
		if tmpConfig.ProjectID.Valid {
			conf.ProjectID = tmpConfig.ProjectID
		}
		if tmpConfig.Name.Valid {
			conf.Name = tmpConfig.Name
		}
		if tmpConfig.Token.Valid {
			conf.Token = tmpConfig.Token
		}
	}
	return nil
}

// GetConsolidatedConfig combines the default config values with the JSON config
// values and environment variables and returns the final result.
func GetConsolidatedConfig(
	jsonRawConf json.RawMessage, env map[string]string, configArg string, external map[string]json.RawMessage,
) (Config, error) {
	result := NewConfig()
	if jsonRawConf != nil {
		jsonConf := Config{}
		if err := json.Unmarshal(jsonRawConf, &jsonConf); err != nil {
			return result, err
		}
		result = result.Apply(jsonConf)
	}
	if err := MergeFromExternal(external, &result); err != nil {
		return result, err
	}

	envConfig := Config{}
	if err := envconfig.Process("", &envConfig); err != nil {
		// TODO: get rid of envconfig and actually use the env parameter...
		return result, err
	}
	result = result.Apply(envConfig)

	if configArg != "" {
		result.Name = null.StringFrom(configArg)
	}

	return result, nil
}
