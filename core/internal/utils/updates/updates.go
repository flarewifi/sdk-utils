package updates

import (
	rpc "core/internal/rpc"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

type CoreReleaseUpdate struct {
	Version        *semver.Version
	CoreZipFileUrl string
	ArchBinFileUrl string
	HasUpdate      bool
}

func CheckCoreReleaseUpdate(currentVersion *semver.Version) (*CoreReleaseUpdate, error) {
	srv, ctx := rpc.GetCoreTwirpServiceAndCtx()

	result, err := srv.FetchLatestCoreRelease(ctx, &rpc.FetchLatestCoreReleaseRequest{
		CurrentCoreVersion: currentVersion.String(),
	})
	if err != nil {
		return nil, err
	}

	if !result.HasNewUpdate {
		return &CoreReleaseUpdate{HasUpdate: false}, nil
	}

	latestVersion, err := semver.NewVersion(fmt.Sprintf("%d.%d.%d", result.GetMajor(), result.GetMinor(), result.GetPatch()))
	if err != nil {
		return nil, err
	}

	update := &CoreReleaseUpdate{
		HasUpdate: true,
		Version:   latestVersion,
	}

	return update, nil
}
