package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/api/config/versions"
	"github.com/monitoror/monitoror/internal/pkg/monitorable/params"
	coreModels "github.com/monitoror/monitoror/models"
	httpApi "github.com/monitoror/monitoror/monitorables/http/api"
	httpModels "github.com/monitoror/monitoror/monitorables/http/api/models"

	"github.com/stretchr/testify/assert"
)

type mockGeneratorParams struct {
	params.Default
	URL string `json:"url"`
}

func TestUsecase_Hydrate(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "EMPTY" },
    { "type": "HTTP-STATUS", "params": { "url": "https://aserver.com" } },
    { "type": "HTTP-RAW", "params": { "url": "https://bserver.com", "regex": "(\\d+)" } },
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "HTTP-STATUS", "params": { "url": "https://aserver.com" } },
      { "type": "HTTP-RAW", "params": { "url": "https://bserver.com" } }
    ]},
		{ "type": "HTTP-FORMATTED", "params": { "url": "https://cserver.com", "format": "JSON", "key": ".key" } },
		{ "type": "HTTP-FORMATTED", "configVariant": "variant1", "params": { "url": "https://cserver.com", "format": "JSON", "key": ".key" } },
    { "type": "HTTP-STATUS", "params": { "url": "https://dserver.com", "statusCodeMin": 200, "statusCodeMax": 399 } }
  ]
}
`

	usecase := initConfigUsecase(nil)
	formattedTileEnabler := usecase.registry.RegisterTile(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName, "variant1"})
	formattedTileEnabler.Enable(coreModels.DefaultVariantName, &httpModels.HTTPFormattedParams{}, "/http/default/formatted")
	formattedTileEnabler.Enable("variant1", &httpModels.HTTPFormattedParams{}, "/http/variant1/formatted")

	config, err := readConfig(input)

	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	assert.Equal(t, "/http/default/status?url=https%3A%2F%2Faserver.com", config.Config.Tiles[1].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[1].InitialMaxDelay)
	assert.Equal(t, "/http/default/raw?regex=%28%5Cd%2B%29&url=https%3A%2F%2Fbserver.com", config.Config.Tiles[2].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[2].InitialMaxDelay)

	group := config.Config.Tiles[3].Tiles
	assert.Equal(t, "/http/default/status?url=https%3A%2F%2Faserver.com", group[0].URL)
	assert.Equal(t, 1000, *group[0].InitialMaxDelay)
	assert.Equal(t, "/http/default/raw?url=https%3A%2F%2Fbserver.com", group[1].URL)
	assert.Equal(t, 1000, *group[1].InitialMaxDelay)

	assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fcserver.com", config.Config.Tiles[4].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[4].InitialMaxDelay)
	assert.Equal(t, "/http/variant1/formatted?format=JSON&key=.key&url=https%3A%2F%2Fcserver.com", config.Config.Tiles[5].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[5].InitialMaxDelay)
	assert.Equal(t, "/http/default/status?statusCodeMax=399&statusCodeMin=200&url=https%3A%2F%2Fdserver.com", config.Config.Tiles[6].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[6].InitialMaxDelay)
}

func TestUsecase_Hydrate_WithGenerator(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}},
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "HTTP-STATUS", "params": { "url": "https://aserver.com" } },
			{ "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:HTTP-FORMATTED", "label": "Test Label", "params": {"url": "https://cserver.com"}}
    ]}
  ]
}
`

	params := &httpModels.HTTPFormattedParams{URL: "https://cserver.com", Format: "JSON", Key: ".key"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)

	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	assert.Equal(t, 4, len(config.Config.Tiles))
	assert.Equal(t, httpApi.HTTPFormattedTileType, config.Config.Tiles[0].Type)
	assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fcserver.com", config.Config.Tiles[0].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[0].InitialMaxDelay)
	assert.Equal(t, "", config.Config.Tiles[2].Tiles[0].Label)
	assert.Equal(t, httpApi.HTTPFormattedTileType, config.Config.Tiles[1].Tiles[1].Type)
	assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fcserver.com", config.Config.Tiles[1].Tiles[1].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[1].Tiles[1].InitialMaxDelay)
	assert.Equal(t, "", config.Config.Tiles[2].Tiles[0].Label)
	assert.Equal(t, httpApi.HTTPFormattedTileType, config.Config.Tiles[2].Tiles[0].Type)
	assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fcserver.com", config.Config.Tiles[2].Tiles[0].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[2].Tiles[0].InitialMaxDelay)
	assert.Equal(t, "", config.Config.Tiles[2].Tiles[0].Label)
	assert.Equal(t, httpApi.HTTPFormattedTileType, config.Config.Tiles[3].Tiles[0].Type)
	assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fcserver.com", config.Config.Tiles[3].Tiles[0].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[3].Tiles[0].InitialMaxDelay)
	assert.Equal(t, "Test Label", config.Config.Tiles[3].Tiles[0].Label)
}

func TestUsecase_Hydrate_WithGeneratorEmpty(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "HTTP-STATUS", "params": { "url": "https://aserver.com" } },
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}}
    ]},
    { "type": "HTTP-STATUS", "params": { "url": "https://bserver.com" } }
  ]
}
`
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) { return []models.GeneratedTile{}, nil }

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)

	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	assert.Equal(t, 2, len(config.Config.Tiles))
}

func TestUsecase_Hydrate_WithGenerator_WithError(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}},
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "HTTP-STATUS", "params": { "url": "https://aserver.com" } },
			{ "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:HTTP-FORMATTED", "configVariant": "variant1", "params": {"url": "https://cserver.com"}}
    ]}
  ]
}
`

	params := &httpModels.HTTPFormattedParams{URL: "https://cserver.com", Format: "JSON", Key: ".key"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}
	mockBuilder2 := func(_ interface{}) ([]models.GeneratedTile, error) { return nil, errors.New("unable to fetch url") }

	usecase := initConfigUsecase(nil)
	tileGeneratorEnabler := usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName, "variant1"})
	tileGeneratorEnabler.Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)
	tileGeneratorEnabler.Enable("variant1", &mockGeneratorParams{}, mockBuilder2)

	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 1)
	assert.Equal(t, config.Errors[0].ID, models.ConfigErrorUnableToHydrate)
	assert.Contains(t, config.Errors[0].Data.ConfigExtract, `GENERATE:HTTP-FORMATTED`)
}

func TestUsecase_Hydrate_WithGenerator_WithTimeoutError(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}},
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "HTTP-STATUS", "params": { "url": "https://aserver.com" } },
			{ "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:HTTP-FORMATTED", "configVariant": "variant1", "params": {"url": "https://cserver.com"}}
    ]}
  ]
}
`

	params := &httpModels.HTTPFormattedParams{URL: "https://cserver.com", Format: "JSON", Key: ".key"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}
	mockBuilder2 := func(_ interface{}) ([]models.GeneratedTile, error) { return nil, context.DeadlineExceeded }

	usecase := initConfigUsecase(nil)
	tileGeneratorEnabler := usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName, "variant1"})
	tileGeneratorEnabler.Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)
	tileGeneratorEnabler.Enable("variant1", &mockGeneratorParams{}, mockBuilder2)
	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 1)
	assert.Equal(t, config.Errors[0].ID, models.ConfigErrorUnableToHydrate)
	assert.Contains(t, config.Errors[0].Data.ConfigExtract, `GENERATE:HTTP-FORMATTED`)
}

func TestUsecase_Hydrate_WithGenerator_WithTimeoutCache(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}}
	]
}
`
	usecase := initConfigUsecase(nil)

	params := &httpModels.HTTPFormattedParams{URL: "https://cserver.com", Format: "JSON", Key: ".key"}
	cachedResult := []models.GeneratedTile{{Params: params}}
	cacheKey := fmt.Sprintf("%s:%s_%s_%s", TileGeneratorStoreKeyPrefix, "GENERATE:HTTP-FORMATTED", "default", `{"url":"https://cserver.com"}`)
	_ = usecase.generatorTileStore.Add(cacheKey, cachedResult, 0)

	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) { return nil, context.DeadlineExceeded }
	usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)

	config, err := readConfig(input)
	if assert.NoError(t, err) {
		usecase.Hydrate(config)
		assert.Len(t, config.Errors, 0)
		assert.Equal(t, httpApi.HTTPFormattedTileType, config.Config.Tiles[0].Type)
		assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fcserver.com", config.Config.Tiles[0].URL)
	}
}

func TestUsecase_Hydrate_TwoGenerators(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://cserver.com"}},
    { "type": "GENERATE:HTTP-FORMATTED", "params": {"url": "https://dserver.com"}}
  ]
}
`
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{
			{Params: &httpModels.HTTPFormattedParams{URL: "https://aserver.com", Format: "JSON", Key: ".key"}},
			{Params: &httpModels.HTTPFormattedParams{URL: "https://bserver.com", Format: "JSON", Key: ".key"}},
		}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)
	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	if assert.Equal(t, 4, len(config.Config.Tiles)) {
		assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Faserver.com", config.Config.Tiles[0].URL)
		assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fbserver.com", config.Config.Tiles[1].URL)
		assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Faserver.com", config.Config.Tiles[2].URL)
		assert.Equal(t, "/http/default/formatted?format=JSON&key=.key&url=https%3A%2F%2Fbserver.com", config.Config.Tiles[3].URL)
	}

}
