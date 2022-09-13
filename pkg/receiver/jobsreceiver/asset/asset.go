package asset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sensu/sensu-go/util/environment"
)

const (
	binDir     = "bin"
	libDir     = "lib"
	includeDir = "include"
)

// A RuntimeAsset is a locally expanded Asset.
type RuntimeAsset struct {
	// Name is the name of the asset
	Name string `mapstructure:"name"`
	// Path is the absolute path to the asset's base directory.
	Path string `mapstructure:"path"`
	// Url is the remote address used for fetching the asset
	Url string `mapstructure:"url"`
	// SHA512 is the hash of the asset tarball.
	SHA512 string `mapstructure:"sha512"`
}

// BinDir returns the full path to the asset's bin directory.
func (r *RuntimeAsset) BinDir() string {
	return filepath.Join(r.Path, binDir)
}

// LibDir returns the full path to the asset's lib directory.
func (r *RuntimeAsset) LibDir() string {
	return filepath.Join(r.Path, libDir)
}

// IncludeDir returns the full path to the asset's include directory.
func (r *RuntimeAsset) IncludeDir() string {
	return filepath.Join(r.Path, includeDir)
}

// Env returns a list of environment variables (e.g. 'PATH=...', 'CPATH=...')
// with asset-specific paths prepended to the parent environment paths for
// each variable, allowing an asset to be used during execution.
func (r *RuntimeAsset) Env() []string {
	assetEnv := []string{
		fmt.Sprintf("PATH=%s${PATH}", r.joinPaths((*RuntimeAsset).BinDir)),
		fmt.Sprintf("LD_LIBRARY_PATH=%s${LD_LIBRARY_PATH}", r.joinPaths((*RuntimeAsset).LibDir)),
		fmt.Sprintf("CPATH=%s${CPATH}", r.joinPaths((*RuntimeAsset).IncludeDir)),
	}
	for i, envVar := range assetEnv {
		// ExpandEnv replaces ${var} with the contents of var from the current
		// environment, or an empty string if var doesn't exist.
		assetEnv[i] = os.ExpandEnv(envVar)
	}

	assetEnv = append(assetEnv, fmt.Sprintf("%s=%s",
		fmt.Sprintf("%s_PATH", environment.Key(r.Name)),
		r.Path,
	))

	return assetEnv
}

// joinPaths joins all paths of a given type for each asset in RuntimeAssetSet.
func (r *RuntimeAsset) joinPaths(pathFunc func(*RuntimeAsset) string) string {
	var sb strings.Builder
	sb.WriteString(pathFunc(r))
	sb.WriteRune(os.PathListSeparator)
	return sb.String()
}

// Install fetches, validates, and extracts the asset.
func (r *RuntimeAsset) Install() error {
	r.Path = filepath.Join(os.TempDir(), r.Name)

	_, err := os.Stat(r.Path)
	if !os.IsNotExist(err) {
		return nil
	}

	var headers map[string]string

	hf := &httpFetcher{}

	f, err := hf.Fetch(context.TODO(), r.Url, headers)

	if err != nil {
		return err
	}

	v := &Sha512Verifier{}
	err = v.Verify(f, r.SHA512)

	if err != nil {
		return err
	}

	e := &archiveExpander{}

	return e.Expand(f, r.Path)
}
