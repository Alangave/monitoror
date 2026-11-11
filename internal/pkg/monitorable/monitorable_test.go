package monitorable

import (
	"os"
	"testing"

	coreModels "github.com/monitoror/monitoror/models"
	httpConfig "github.com/monitoror/monitoror/monitorables/http/config"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	_ = os.Setenv("MO_MONITORABLE_HTTP_TIMEOUT", "1337")
	_ = os.Setenv("MO_MONITORABLE_HTTP_TEST_TIMEOUT", "1337")

	conf := make(map[coreModels.VariantName]httpConfig.HTTP)
	LoadConfig(&conf, httpConfig.Default)

	assert.Len(t, conf, 2)
	assert.Equal(t, 1337, conf["default"].Timeout)
	assert.Equal(t, 1337, conf["test"].Timeout)

}

func TestValidateConfig(t *testing.T) {
	httpConf := &httpConfig.HTTP{Timeout: -1}
	errors := ValidateConfig(httpConf, "test")
	assert.NotEmpty(t, errors)
	assert.Equal(t, `Invalid "MO_MONITORABLE_HTTP_TEST_TIMEOUT" field. Must be greater or equal to 0.`, errors[0].Error())

	httpConf.Timeout = 2000
	errors = ValidateConfig(httpConf, "test")
	assert.Empty(t, errors)
}

func TestGetVariantsNames(t *testing.T) {
	conf := map[coreModels.VariantName]httpConfig.HTTP{
		"test": {},
	}

	variants := GetVariantsNames(conf)
	assert.Len(t, variants, 1)
	assert.Equal(t, coreModels.VariantName("test"), variants[0])
	assert.Panics(t, func() { GetVariantsNames("test") })
}

func TestBuildMonitorableEnvKey(t *testing.T) {
	assert.Equal(t, "MO_MONITORABLE_HTTP_TEST_TEST", buildMonitorableEnvKey(httpConfig.Default, "test", "test"))
	assert.Equal(t, "MO_MONITORABLE_HTTP_TEST", buildMonitorableEnvKey(httpConfig.Default, coreModels.DefaultVariantName, "test"))
	assert.Panics(t, func() {
		buildMonitorableEnvKey("test", "test", "test")
	})
}
