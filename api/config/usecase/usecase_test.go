package usecase

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/monitoror/monitoror/api/config"
	"github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/api/config/repository"
	"github.com/monitoror/monitoror/api/config/versions"
	coreConfig "github.com/monitoror/monitoror/config"
	coreModels "github.com/monitoror/monitoror/models"
	httpApi "github.com/monitoror/monitoror/monitorables/http/api"
	httpModels "github.com/monitoror/monitoror/monitorables/http/api/models"
	"github.com/monitoror/monitoror/registry"
	"github.com/monitoror/monitoror/store"

	"github.com/jsdidierlaurent/echo-middleware/cache"
	"github.com/stretchr/testify/assert"
)

func initConfigUsecase(repository config.Repository) *configUsecase {
	s := &store.Store{
		CoreConfig: &coreConfig.CoreConfig{InitialMaxDelay: 1000},
		CacheStore: cache.NewGoCacheStore(time.Second, time.Second),
		Registry:   registry.NewRegistry(),
	}

	usecase := NewConfigUsecase(repository, s).(*configUsecase)

	usecase.registry.RegisterTile(httpApi.HTTPStatusTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &httpModels.HTTPStatusParams{}, "/http/default/status")
	usecase.registry.RegisterTile(httpApi.HTTPRawTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName, "disabledVariant"}).
		Enable(coreModels.DefaultVariantName, &httpModels.HTTPRawParams{}, "/http/default/raw")
	usecase.registry.RegisterTile(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &httpModels.HTTPFormattedParams{}, "/http/default/formatted")

	return usecase
}

func readConfig(input string) (configBag *models.ConfigBag, err error) {
	configBag = &models.ConfigBag{}
	reader := io.NopCloser(strings.NewReader(input))
	configBag.Config, err = repository.ReadConfig(reader)
	return
}

func TestUsecase_Global_Success(t *testing.T) {
	rawConfig := fmt.Sprintf(`
{
	"version" : %q,
  "columns": 4,
  "tiles": [
		{ "type": "HTTP-STATUS", "label": "Monitoror", "params": {"url": "https://monitoror.com"} }
  ]
}
`, versions.CurrentVersion)

	expectRawConfig := fmt.Sprintf(`{"config":{"version":%q,"columns":4,"tiles":[{"type":"HTTP-STATUS","label":"Monitoror","url":"/http/default/status?url=https%%3A%%2F%%2Fmonitoror.com","initialMaxDelay":1000}]}}`, versions.CurrentVersion)

	config, err := readConfig(rawConfig)
	if assert.NoError(t, err) {
		usecase := initConfigUsecase(nil)
		usecase.Verify(config)
		usecase.Hydrate(config)

		assert.Len(t, config.Errors, 0)

		marshal, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.Equal(t, expectRawConfig, string(marshal))
	}
}
