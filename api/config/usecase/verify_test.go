package usecase

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/api/config/versions"
	"github.com/monitoror/monitoror/internal/pkg/validator"
	coreModels "github.com/monitoror/monitoror/models"
	httpApi "github.com/monitoror/monitoror/monitorables/http/api"
	httpModels "github.com/monitoror/monitoror/monitorables/http/api/models"

	"github.com/stretchr/testify/assert"
)

func initConfig(t *testing.T, rawConfig string) (tiles *models.TileConfig, conf *models.ConfigBag) {
	tiles = &models.TileConfig{}

	err := json.Unmarshal([]byte(rawConfig), &tiles)
	assert.NoError(t, err)

	conf = &models.ConfigBag{Config: &models.Config{Version: versions.CurrentVersion.ToConfigVersion()}}

	return
}

func TestUsecase_Verify_Success(t *testing.T) {
	rawConfig := fmt.Sprintf(`
{
	"version" : %q,
  "columns": 4,
	"zoom": 2.5,
  "tiles": [
		{ "type": "EMPTY" }
  ]
}
`, versions.CurrentVersion)

	conf, err := readConfig(rawConfig)
	if assert.NoError(t, err) {
		usecase := initConfigUsecase(nil)
		usecase.Verify(conf)

		assert.Len(t, conf.Errors, 0)
	}
}

func TestUsecase_Verify_SuccessWithOptionalParameters(t *testing.T) {
	rawConfig := fmt.Sprintf(`
{
	"version" : %q,
  "columns": 4,
  "zoom": 2.5,
  "tiles": [
		{ "type": "EMPTY" }
  ]
}
`, versions.CurrentVersion)

	conf, err := readConfig(rawConfig)

	if assert.NoError(t, err) {
		usecase := initConfigUsecase(nil)
		usecase.Verify(conf)

		assert.Len(t, conf.Errors, 0)
	}
}

func TestUsecase_Verify_Failed(t *testing.T) {
	for _, testcase := range []struct {
		rawConfig string
		errorID   models.ConfigErrorID
		errorData models.ConfigErrorData
	}{
		{
			rawConfig: `{}`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{FieldName: "version"},
		},
		{
			rawConfig: `{"version": "0.0"}`,
			errorID:   models.ConfigErrorUnsupportedVersion,
			errorData: models.ConfigErrorData{
				FieldName: "version",
				Value:     `"0.0"`,
				Expected:  fmt.Sprintf(`%q <= version <= %q`, versions.MinimalVersion, versions.CurrentVersion),
			},
		},
		{
			rawConfig: `{"version": "999.999"}`, // Don't let me go that far ^^'
			errorID:   models.ConfigErrorUnsupportedVersion,
			errorData: models.ConfigErrorData{
				FieldName: "version",
				Value:     `"999.999"`,
				Expected:  fmt.Sprintf(`%q <= version <= %q`, versions.MinimalVersion, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "columns",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 0, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "columns",
				Expected:      "columns > 0",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":0,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1, "zoom": 0, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "zoom",
				Expected:      "zoom > 0",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1,"zoom":0,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1, "zoom": 19.8, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "zoom",
				Expected:      "zoom <= 10",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1,"zoom":19.8,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1, "tiles": []}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1,"tiles":[]}`, versions.CurrentVersion),
			},
		},
	} {
		conf, err := readConfig(testcase.rawConfig)
		if assert.NoError(t, err) {
			usecase := initConfigUsecase(nil)
			usecase.Verify(conf)
			if assert.Len(t, conf.Errors, 1) {
				assert.Equal(t, testcase.errorID, conf.Errors[0].ID)
				assert.Equal(t, testcase.errorData, conf.Errors[0].Data)
			}
		}
	}
}

func TestUsecase_VerifyTile_Success(t *testing.T) {
	rawConfig := `{ "type": "HTTP-STATUS", "columnSpan": 2, "rowSpan": 2, "params": { "url": "https://bserver.com" } }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_Success_Empty(t *testing.T) {
	rawConfig := `{ "type": "EMPTY" }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_Success_Group(t *testing.T) {
	rawConfig := `
      { "type": "GROUP", "label": "...", "tiles": [
          { "type": "HTTP-STATUS", "params": { "url": "https://aserver.com" } },
          { "type": "HTTP-RAW", "params": { "url": "https://bserver.com" } }
			]}
`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_Failed(t *testing.T) {
	for _, testcase := range []struct {
		rawConfig string
		errorID   models.ConfigErrorID
		errorData models.ConfigErrorData
	}{
		{
			rawConfig: `{ "type": "HTTP-STATUS", "columnSpan": -1, "params": { "url": "https://server.com" } }`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "columnSpan",
				Expected:      "columnSpan > 0",
				ConfigExtract: `{"type":"HTTP-STATUS","columnSpan":-1,"params":{"url":"https://server.com"}}`,
			},
		},
		{
			rawConfig: `{ "type": "HTTP-STATUS", "rowSpan": -1, "params": { "url": "https://server.com" } }`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "rowSpan",
				Expected:      "rowSpan > 0",
				ConfigExtract: `{"type":"HTTP-STATUS","rowSpan":-1,"params":{"url":"https://server.com"}}`,
			},
		},
		{
			rawConfig: `
					{ "type": "GROUP", "tiles": [
							{ "type": "EMPTY" }
					]}
`,
			errorID: models.ConfigErrorUnauthorizedSubtileType,
			errorData: models.ConfigErrorData{
				ConfigExtract:          `{"type":"GROUP","tiles":[{"type":"EMPTY"}]}`,
				ConfigExtractHighlight: `{"type":"EMPTY"}`,
			},
		},
		{
			rawConfig: `
					{ "type": "GROUP", "tiles": [
							{ "type": "GROUP" }
					]}
`,
			errorID: models.ConfigErrorUnauthorizedSubtileType,
			errorData: models.ConfigErrorData{
				ConfigExtract:          `{"type":"GROUP","tiles":[{"type":"GROUP"}]}`,
				ConfigExtractHighlight: `{"type":"GROUP"}`,
			},
		},
		{
			rawConfig: `{ "type": "GROUP", "params": {"test": "test"}}`,
			errorID:   models.ConfigErrorUnauthorizedField,
			errorData: models.ConfigErrorData{
				FieldName:     "params",
				ConfigExtract: `{"type":"GROUP","params":{"test":"test"}}`,
			},
		},
		{
			rawConfig: `{ "type": "GROUP"}`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: `{"type":"GROUP"}`,
			},
		},
		{
			rawConfig: `{ "type": "GROUP", "tiles": []}`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: `{"type":"GROUP"}`,
			},
		},
		{
			rawConfig: `{ "type": "HTTP-STATUS" }`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "params",
				ConfigExtract: `{"type":"HTTP-STATUS","configVariant":"default"}`,
			},
		},
		{
			rawConfig: `{ "type": "HTTP-STATUS", "params": { } }`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "url",
				ConfigExtract: `{"type":"HTTP-STATUS","configVariant":"default","params":{}}`,
			},
		},
		{
			rawConfig: `{ "type": "HTTP-STATUS", "params": { "host": "https://server.com" } }`,
			errorID:   models.ConfigErrorUnknownField,
			errorData: models.ConfigErrorData{
				FieldName:     "host",
				ConfigExtract: `{"type":"HTTP-STATUS","params":{"host":"https://server.com"},"configVariant":"default"}`,
				Expected:      "url",
			},
		},
		{
			rawConfig: `{ "type": "HTTP-STATUS", "params": { "url": "server.com" } }`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "url",
				ConfigExtract: `{"type":"HTTP-STATUS","params":{"url":"server.com"},"configVariant":"default"}`,
			},
		},
		{
			rawConfig: `{ "type": "HTTP-FORMATTED", "params": { "url": "https://server.com", "format": "TOTO", "key": ".key" } }`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "format",
				Expected:      "JSON, YAML, XML",
				ConfigExtract: `{"type":"HTTP-FORMATTED","params":{"format":"TOTO","key":".key","url":"https://server.com"},"configVariant":"default"}`,
			},
		},
		{
			rawConfig: `{ "type": "HTTP-STATUS", "params": { "url": ["https://server.com"] } }`,
			errorID:   models.ConfigErrorUnexpectedError,
			errorData: models.ConfigErrorData{
				FieldName:     "params",
				ConfigExtract: `{"type":"HTTP-STATUS","params":{"url":["https://server.com"]},"configVariant":"default"}`,
			},
		},
		{
			rawConfig: `{ "type": "HTTP-RAW", "configVariant": "disabledVariant", "params": { } }`,
			errorID:   models.ConfigErrorDisabledVariant,
			errorData: models.ConfigErrorData{
				FieldName:     "configVariant",
				Value:         `"disabledVariant"`,
				ConfigExtract: `{"type":"HTTP-RAW","configVariant":"disabledVariant"}`,
			},
		},
	} {
		tile, conf := initConfig(t, testcase.rawConfig)
		usecase := initConfigUsecase(nil)
		usecase.verifyTile(conf, tile, nil)

		if assert.Len(t, conf.Errors, 1) {
			assert.Equal(t, testcase.errorID, conf.Errors[0].ID)
			assert.Equal(t, testcase.errorData, conf.Errors[0].Data)
		}
	}
}

func TestUsecase_VerifyTile_Failed_WrongMinimalVerison(t *testing.T) {
	rawConfig := `{ "type": "HTTP-STATUS", "params": { "url": "https://server.com" } }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.registry.TileMetadata["HTTP-STATUS"].MinimalVersion = "999.0"
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnsupportedTileInThisVersion, conf.Errors[0].ID)
		assert.Equal(t, "type", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `{"type":"HTTP-STATUS","params":{"url":"https://server.com"},"configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
		assert.Equal(t, `version >= "999.0"`, conf.Errors[0].Data.Expected)
	}
}

func TestUsecase_VerifyTile_Failed_WrongTileType(t *testing.T) {
	rawConfig := `{ "type": "PONG", "params": { "url": "https://server.com" } }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownTileType, conf.Errors[0].ID)
		assert.Equal(t, "type", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `{"type":"PONG","params":{"url":"https://server.com"},"configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
	}
}

func TestUsecase_VerifyTile_WithGenerator(t *testing.T) {
	rawConfig := `{ "type": "GENERATE:HTTP-FORMATTED", "configVariant": "default", "params": { "url": "https://cserver.com" } }`

	tile, conf := initConfig(t, rawConfig)
	params := &httpModels.HTTPFormattedParams{URL: "https://cserver.com", Format: "JSON", Key: ".key"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_WithGenerator_WithWrongGenerator(t *testing.T) {
	rawConfig := `{ "type": "GENERATE:HTTP-STATUS", "params": {}}`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, nil)

	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownGeneratorTileType, conf.Errors[0].ID)
		assert.Equal(t, "type", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `{"type":"GENERATE:HTTP-STATUS","configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
		assert.Equal(t, `GENERATE:HTTP-FORMATTED`, conf.Errors[0].Data.Expected)
	}
}

func TestUsecase_VerifyTile_WithWrongVariant(t *testing.T) {
	rawConfig := `{ "type": "HTTP-RAW", "configVariant": "test", "params": { "url": "https://server.com" } }`

	tile, conf := initConfig(t, rawConfig)

	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownVariant, conf.Errors[0].ID)
		assert.Equal(t, "configVariant", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `"test"`, conf.Errors[0].Data.Value)
		assert.Contains(t, conf.Errors[0].Data.Expected, coreModels.DefaultVariantName)
		assert.Contains(t, conf.Errors[0].Data.Expected, "disabledVariant")
		assert.Equal(t, `{"type":"HTTP-RAW","params":{"url":"https://server.com"},"configVariant":"test"}`, conf.Errors[0].Data.ConfigExtract)
	}
}

func TestUsecase_VerifyTile_WithGenerator_WithWrongVariant(t *testing.T) {
	rawConfig := `{ "type": "GENERATE:HTTP-FORMATTED", "configVariant": "test", "params": { "url": "https://cserver.com" } }`

	tile, conf := initConfig(t, rawConfig)
	params := &httpModels.HTTPFormattedParams{URL: "https://cserver.com", Format: "JSON", Key: ".key"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(httpApi.HTTPFormattedTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &mockGeneratorParams{}, mockBuilder)
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownVariant, conf.Errors[0].ID)
		assert.Equal(t, "configVariant", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `"test"`, conf.Errors[0].Data.Value)
		assert.Contains(t, conf.Errors[0].Data.Expected, coreModels.DefaultVariantName)
		assert.Equal(t, `{"type":"GENERATE:HTTP-FORMATTED","params":{"url":"https://cserver.com"},"configVariant":"test"}`, conf.Errors[0].Data.ConfigExtract)
	}
}

type minimalVersionTest struct {
	Field1 string `json:"field1" available:"since=999.0"`
}

func (s *minimalVersionTest) Validate() []validator.Error { return nil }

func TestUsecase_VerifyTile_FieldMinimalVersion(t *testing.T) {
	rawConfig := `{ "type": "TEST", "params": { "field1": "server.com" } }`
	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterTile("TEST", versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &minimalVersionTest{}, "/test/default/test")

	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnsupportedTileParamInThisVersion, conf.Errors[0].ID)
		assert.Equal(t, "field1", conf.Errors[0].Data.FieldName)
		assert.Equal(t, "version >= 999.0", conf.Errors[0].Data.Expected)
		assert.Equal(t, `{"type":"TEST","params":{"field1":"server.com"},"configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
	}
}
