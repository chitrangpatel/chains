/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	cm "knative.dev/pkg/configmap"
)

type Config struct {
	Artifacts    ArtifactConfigs
	Storage      StorageConfigs
	Signers      SignerConfigs
	Builder      BuilderConfig
	Transparency TransparencyConfig
}

// ArtifactConfig contains the configuration for how to sign/store/format the signatures for each artifact type
type ArtifactConfigs struct {
	TaskRuns Artifact
	OCI      Artifact
}

// Artifact contains the configuration for how to sign/store/format the signatures for a single artifact
type Artifact struct {
	Format         string
	StorageBackend string
	Signer         string
}

// StorageConfig contains the configuration to instantiate different storage providers
type StorageConfigs struct {
	GCS    GCSStorageConfig
	OCI    OCIStorageConfig
	Tekton TektonStorageConfig
	DocDB  DocDBStorageConfig
}

// SigningConfig contains the configuration to instantiate different signers
type SignerConfigs struct {
	X509 X509Signer
	KMS  KMSSigner
}

type BuilderConfig struct {
	ID string
}

type X509Signer struct {
	FulcioEnabled bool
	FulcioAuth    string
}

type KMSSigner struct {
	KMSRef string
}

type GCSStorageConfig struct {
	Bucket string
}

type OCIStorageConfig struct {
	Repository string
	Insecure   bool
}

type TektonStorageConfig struct {
}

type DocDBStorageConfig struct {
	URL string
}

type TransparencyConfig struct {
	Enabled bool
	URL     string
}

const (
	taskrunFormatKey  = "artifacts.taskrun.format"
	taskrunStorageKey = "artifacts.taskrun.storage"
	taskrunSignerKey  = "artifacts.taskrun.signer"

	ociFormatKey  = "artifacts.oci.format"
	ociStorageKey = "artifacts.oci.storage"
	ociSignerKey  = "artifacts.oci.signer"

	gcsBucketKey             = "storage.gcs.bucket"
	ociRepositoryKey         = "storage.oci.repository"
	ociRepositoryInsecureKey = "storage.oci.repository.insecure"
	docDBUrlKey              = "storage.docdb.url"
	// No config needed for Tekton object storage

	// No config needed for x509 signer

	// KMS
	kmsSignerKMSRef = "signers.kms.kmsref"
	// Fulcio
	x509SignerFulcioEnabled = "signers.x509.fulcio.enabled"
	x509SignerFulcioAuth    = "signers.x509.fulcio.auth"

	// Builder config
	builderIDKey = "builder.id"

	transparencyEnabledKey = "transparency.enabled"
	transparencyURLKey     = "transparency.url"

	ChainsConfig = "chains-config"
)

func defaultConfig() *Config {
	return &Config{
		Artifacts: ArtifactConfigs{
			TaskRuns: Artifact{
				Format:         "tekton",
				StorageBackend: "tekton",
				Signer:         "x509",
			},
			OCI: Artifact{
				Format:         "simplesigning",
				StorageBackend: "oci",
				Signer:         "x509",
			},
		},
		Transparency: TransparencyConfig{
			URL: "https://rekor.sigstore.dev",
		},
		Signers: SignerConfigs{
			X509: X509Signer{
				FulcioAuth: "google",
			},
		},
		Builder: BuilderConfig{
			ID: "tekton-chains",
		},
	}
}

// NewConfigFromMap creates a Config from the supplied map
func NewConfigFromMap(data map[string]string) (*Config, error) {
	cfg := defaultConfig()

	if err := cm.Parse(data,
		// Artifact-specific configs
		// TaskRuns
		asString(taskrunFormatKey, &cfg.Artifacts.TaskRuns.Format, "tekton", "in-toto", "tekton-provenance"),
		asString(taskrunStorageKey, &cfg.Artifacts.TaskRuns.StorageBackend, "tekton", "oci", "gcs", "docdb"),
		asString(taskrunSignerKey, &cfg.Artifacts.TaskRuns.Signer, "x509", "kms"),
		// OCI
		asString(ociFormatKey, &cfg.Artifacts.OCI.Format, "tekton", "simplesigning"),
		asString(ociStorageKey, &cfg.Artifacts.OCI.StorageBackend, "tekton", "oci", "gcs", "docdb"),
		asString(ociSignerKey, &cfg.Artifacts.OCI.Signer, "x509", "kms"),

		// Storage level configs
		asString(gcsBucketKey, &cfg.Storage.GCS.Bucket),
		asString(ociRepositoryKey, &cfg.Storage.OCI.Repository),
		cm.AsBool(ociRepositoryInsecureKey, &cfg.Storage.OCI.Insecure),
		asString(docDBUrlKey, &cfg.Storage.DocDB.URL),

		cm.AsBool(transparencyEnabledKey, &cfg.Transparency.Enabled),
		asString(transparencyURLKey, &cfg.Transparency.URL),

		asString(kmsSignerKMSRef, &cfg.Signers.KMS.KMSRef),

		cm.AsBool(x509SignerFulcioEnabled, &cfg.Signers.X509.FulcioEnabled),
		asString(x509SignerFulcioAuth, &cfg.Signers.X509.FulcioAuth),

		// Build config
		asString(builderIDKey, &cfg.Builder.ID),
	); err != nil {
		return nil, fmt.Errorf("failed to parse data: %w", err)
	}

	return cfg, nil
}

// NewConfigFromConfigMap creates a Config from the supplied ConfigMap
func NewConfigFromConfigMap(configMap *corev1.ConfigMap) (*Config, error) {
	return NewConfigFromMap(configMap.Data)
}

// asString passes the value at key through into the target, if it exists.
// TODO(mattmoor): This might be a nice variation on cm.AsString to upstream.
func asString(key string, target *string, values ...string) cm.ParseFunc {
	return func(data map[string]string) error {
		raw, ok := data[key]
		if !ok {
			return nil
		}
		if len(values) > 0 {
			vals := sets.NewString(values...)
			if !vals.Has(raw) {
				return fmt.Errorf("invalid value %q wanted one of %v", raw, vals.List())
			}
		}
		*target = raw
		return nil
	}
}