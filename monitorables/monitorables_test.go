package monitorables

import (
	"testing"

	"github.com/monitoror/monitoror/internal/pkg/monitorable/test"
	httpApi "github.com/monitoror/monitoror/monitorables/http/api"
	"github.com/monitoror/monitoror/registry"

	"github.com/stretchr/testify/assert"
)

func TestManager_RegisterMonitorables(t *testing.T) {
	// init Store
	store, _ := test.InitMockAndStore()
	store.Registry = registry.NewRegistry()

	RegisterMonitorables(store)

	mr := store.Registry.(*registry.MetadataRegistry)

	// ------------ HTTP ------------
	assert.NotNil(t, mr.TileMetadata[httpApi.HTTPStatusTileType])
	assert.NotNil(t, mr.TileMetadata[httpApi.HTTPRawTileType])
	assert.NotNil(t, mr.TileMetadata[httpApi.HTTPFormattedTileType])
}
