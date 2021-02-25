package version

import (
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	params.VersionMajor = 0     // Major version component of the current release
	params.VersionMinor = 7     // Minor version component of the current release
	params.VersionPatch = 0     // Patch version component of the current release
	params.VersionMeta = "rc.3" // Version metadata to append to the version string
}
