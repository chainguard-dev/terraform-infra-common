package otel

import (
	"context"
	_ "embed"
	"os"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"
)

var (
	//go:embed default.yaml
	defaultConfig string
	// fileprovider parses a YAML config file, and envprovider enables env var substitution
	// inside the same file.
	configProviders = makeConfigProviderMap(fileprovider.New(), envprovider.New())
)

// StartCollectorAsync starts the collector asynchronously.
func StartCollectorAsync(ctx context.Context) (shutdown func(), runErr chan error, err error) {
	collector, err := NewCollector()
	if err != nil {
		return nil, nil, err
	}
	errChan := make(chan error)
	go func() {
		defer close(errChan)
		err := collector.Run(ctx)
		errChan <- err
	}()
	return collector.Shutdown, errChan, nil
}

func NewCollector() (*otelcol.Collector, error) {
	return NewCollectorWithConfig("")
}

func NewCollectorWithConfig(configYAML string) (*otelcol.Collector, error) {
	settings := newDefaultSettings()
	// Create a new temp file and write the default config to it.
	//
	// This is workaround since we don't have a config provider that just take a YAML blob.
	// Once we have that, we can just pass the default config as a string to the config provider.
	configFile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}
	if configYAML == "" {
		configYAML = defaultConfig
	}
	if err := os.WriteFile(configFile.Name(), []byte(configYAML), 0600); err != nil {
		return nil, err
	}
	return newCollectorWithFlags(settings, configFile.Name())
}

func newDefaultSettings() otelcol.CollectorSettings {
	info := component.BuildInfo{
		Command:     "otelcol",
		Description: "cloudrun in-process otel-collector",
	}

	params := otelcol.CollectorSettings{BuildInfo: info, Factories: components}
	return params
}

func makeConfigProviderMap(providers ...confmap.Provider) map[string]confmap.Provider {
	ret := make(map[string]confmap.Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}

func newDefaultConfigProviderSettings(configFile string) otelcol.ConfigProviderSettings {
	settings := otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:       []string{configFile},
			Providers:  configProviders,
			Converters: []confmap.Converter{expandconverter.New()},
		},
	}
	for _, provider := range []confmap.Provider{fileprovider.New(), envprovider.New()} {
		settings.ResolverSettings.Providers[provider.Scheme()] = provider
	}
	return settings
}

func newCollectorWithFlags(set otelcol.CollectorSettings, configFile string) (*otelcol.Collector, error) {
	var err error
	set.ConfigProvider, err = otelcol.NewConfigProvider(newDefaultConfigProviderSettings(configFile))
	if err != nil {
		return nil, err
	}
	return otelcol.NewCollector(set)
}

func components() (otelcol.Factories, error) {
	var err error
	factories := otelcol.Factories{}

	factories.Receivers, err = receiver.MakeFactoryMap(
		prometheusreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Exporters, err = exporter.MakeFactoryMap(
		googlecloudexporter.NewFactory(),
		googlemanagedprometheusexporter.NewFactory(),
		loggingexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Processors, err = processor.MakeFactoryMap(
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		resourcedetectionprocessor.NewFactory(),
		resourceprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	factories.Extensions, err = extension.MakeFactoryMap(
		healthcheckextension.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	return factories, nil
}
