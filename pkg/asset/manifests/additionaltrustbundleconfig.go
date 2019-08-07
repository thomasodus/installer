package manifests

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
)

var (
	additionalTrustBundleConfigFileName = filepath.Join(manifestDir, "user-ca-bundle-config.yaml")
)

const (
	additionalTrustBundleConfigDataKey = "ca-bundle.crt"
	additionalTrustBundleConfigMapName = "user-ca-bundle"
)

// AdditionalTrustBundleConfig generates the additional-trust-bundle-config.yaml files.
type AdditionalTrustBundleConfig struct {
	ConfigMap *corev1.ConfigMap
	File      *asset.File
}

var _ asset.WritableAsset = (*AdditionalTrustBundleConfig)(nil)

// Name returns a human friendly name for the asset.
func (*AdditionalTrustBundleConfig) Name() string {
	return "Additional Trust Bundle Config"
}

// Dependencies returns all of the dependencies directly needed to generate
// the asset.
func (*AdditionalTrustBundleConfig) Dependencies() []asset.Asset {
	return []asset.Asset{
		&installconfig.InstallConfig{},
	}
}

// Generate generates the CloudProviderConfig.
func (atbc *AdditionalTrustBundleConfig) Generate(dependencies asset.Parents) error {
	installConfig := &installconfig.InstallConfig{}
	dependencies.Get(installConfig)

	if installConfig.Config.AdditionalTrustBundle == "" {
		return nil
	}
	data, err := parseCertificates(installConfig.Config.AdditionalTrustBundle)

	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "openshift-config",
			Name:      additionalTrustBundleConfigMapName,
		},
		Data: data,
	}

	cmData, err := yaml.Marshal(cm)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s manifest", atbc.Name())
	}
	atbc.ConfigMap = cm
	atbc.File = &asset.File{
		Filename: additionalTrustBundleConfigFileName,
		Data:     cmData,
	}
	return nil
}

// Files returns the files generated by the asset.
func (atbc *AdditionalTrustBundleConfig) Files() []*asset.File {
	if atbc.File != nil {
		return []*asset.File{atbc.File}
	}
	return []*asset.File{}
}

// Load loads the already-rendered files back from disk.
func (atbc *AdditionalTrustBundleConfig) Load(f asset.FileFetcher) (bool, error) {
	return false, nil
}

func parseCertificates(certificates string) (map[string]string, error) {
	rest := []byte(certificates)
	var sb strings.Builder
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, fmt.Errorf("unable to parse certificate, please check the additionalTrustBundle section of install-config.yaml")
		}

		cert, err := x509.ParseCertificate(block.Bytes)

		if err != nil {
			return nil, err
		}

		if cert.IsCA {
			sb.WriteString(string(pem.EncodeToMemory(block)))
		}

		if len(rest) == 0 {
			break
		}
	}
	return map[string]string{additionalTrustBundleConfigDataKey: sb.String()}, nil
}
