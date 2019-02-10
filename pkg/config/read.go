package config

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/google/go-jsonnet"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	cfgv1 "github.com/mbrt/gmailctl/pkg/config/v1alpha1"
	cfgv2 "github.com/mbrt/gmailctl/pkg/config/v1alpha2"
)

// LatestVersion points to the latest version of the config format.
const LatestVersion = cfgv2.Version

// ReadFile takes a path and returns the parsed config file.
func ReadFile(path string) (cfgv2.Config, error) {
	/* #nosec */
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return cfgv2.Config{}, NotFoundError(err)
	}
	if filepath.Ext(path) == ".jsonnet" {
		return readJsonnet(path, b)
	}
	return readYaml(b)
}

func readJsonnet(path string, buf []byte) (cfgv2.Config, error) {
	var res cfgv2.Config
	vm := jsonnet.MakeVM()
	jstr, err := vm.EvaluateSnippet(path, string(buf))
	if err != nil {
		return res, errors.Wrap(err, "invalid jsonnet")
	}
	err = json.Unmarshal([]byte(jstr), &res)
	if err != nil {
		return res, err
	}
	if res.Version != cfgv2.Version {
		return res, errors.Errorf("unsupported version '%s'", res.Version)
	}
	return res, nil
}

func readYaml(buf []byte) (cfgv2.Config, error) {
	var res cfgv2.Config
	version, err := readVersion(buf)
	if err != nil {
		return res, errors.Wrap(err, "error parsing the config version")
	}

	switch version {
	case cfgv2.Version:
		err = yaml.UnmarshalStrict(buf, &res)
		return res, err

	case cfgv1.Version:
		var v1 cfgv1.Config
		err = yaml.UnmarshalStrict(buf, &v1)
		if err != nil {
			return res, errors.Wrap(err, "error parsing v1alpha1 config")
		}
		return cfgv2.Import(v1)

	default:
		return res, errors.Errorf("unknown config version: %s", version)
	}
}

func readVersion(buf []byte) (string, error) {
	// Try to unmarshal only the version
	v := struct {
		Version string `yaml:"version"`
	}{}
	err := yaml.Unmarshal(buf, &v)
	return v.Version, err
}

// IsNotFound returns true if an error is related to a file not found
func IsNotFound(err error) bool {
	nfErr, ok := errors.Cause(err).(notFound)
	return ok && nfErr.NotFound()
}

// NotFoundError wraps the given error and makes it into a not found one
func NotFoundError(err error) error {
	if err == nil {
		return nil
	}
	return notFoundError{err}
}

type notFound interface {
	NotFound() bool
}

type notFoundError struct {
	error
}

func (e notFoundError) Error() string  { return e.error.Error() }
func (e notFoundError) NotFound() bool { return true }
