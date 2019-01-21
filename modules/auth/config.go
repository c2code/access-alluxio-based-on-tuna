package auth

import (
	"github.com/spf13/viper"
	"hexmeet.com/haishen/tuna/logp"
	"github.com/gin-gonic/gin"
	"encoding/json"
	"net/http"
)

// ModuleName for log
const (
	ModuleName string = "auth"
)

// Content Types
const (
	ContentTypeJSON   = "application/json"
)

// Request Type
const (
	RequestExample                = "RequestExample"
	RequestAlluxioCreateUser      = "RequestAlluxioCreateUser"
	RequestAlluxioDeleteUser      = "RequestAlluxioDeleteUser"
	RequestAlluxioCreateFile      = "RequestAlluxioCreateFile"
	RequestAlluxioWriteContent    = "RequestAlluxioWriteContent"
	RequestAlluxioOpenFile        = "RequestAlluxioOpenFile"
	RequestAlluxioReadContent     = "RequestAlluxioReadContent"
	RequestAlluxioCloseFile       = "RequestAlluxioCloseFile"
	RequestAlluxioDeleteFile      = "RequestAlluxioDeleteFile"
	RequestAlluxioRenameFile      = "RequestAlluxioRenameFile"
)

// API response error code
const (
	ErrCodeOk                  = 0
	ErrCodeGeneral             = 1
	ErrCodeFailedToReadBody    = 2
	ErrCodeFailedToParseBody   = 3
	ErrCodeTimeout             = 4
	ErrCodeUserDeny            = 5
	ErrCodeOpenFail            = 6
	ErrCodeReadFail            = 7
	ErrCodeCreateFileFail      = 8
	ErrCodeWriteFail           = 9
	ErrCodeCreateUserFail      = 10
	ErrCodeDeleteUserFail      = 11
	ErrCodeDeleteFileFail      = 12
	ErrCodeRenameFileFail      = 13
)

// API response error info
const (
	ErrInfoOk                  = "RESULT_OK"
	ErrInfoFailedToReadBody    = "FailedToReadBody"
	ErrInfoFailedToParseBody   = "FailedToParseBody"
	ErrInfoTimeout             = "Timeout"
	ErrInfoUserDeny            = "ErrInfoUserDeny"
	ErrInfoOpenFail            = "ErrInfoOpenFail"
	ErrInfoReadFail            = "ErrInfoReadFail"
	ErrInfoCreateFileFail      = "ErrInfoCreateFail"
	ErrInfoWriteFail           = "ErrInfoWriteFail"
	ErrInfoCreateUserFail      = "ErrInfoCreateUserFail"
	ErrInfoDeleteUserFail      = "ErrInfoDeleteUserFail"
	ErrInfoDeleteFileFail      = "ErrInfoDeleteFileFail"
	ErrInfoRenameFileFail      = "ErrInfoRenameFileFail"
)

// BaseResponse definition
type BaseResponse struct {
	ErrCode  int    `json:"err_code"`
	ErrInfo  string `json:"err_info"`
	MoreInfo string `json:"more_info"`
}

// Config config for audit manager
type Config struct {
	MaxWorker    int    `json:"maxworker"`
	WebPort      int    `json:"webport"`
	ReqTimeout   int    `json:"reqtimeout"`
	Debug        bool   `json:"debug"`
}

type GetLogLevelResponse struct {
	BaseResponse
	Level string        `json:"level"`
}

// SetLevelRequest definition
type SetLogLevelRequest struct {
	Level string `json:"level"`
}

var defaultConfig = Config {
	MaxWorker:  20,
	WebPort:    8088,
	ReqTimeout: 10000,
	Debug:      false,
}

//get default config
func DefaultConfig() Config{
	return defaultConfig
}

//load config file : tuna.json
func initConfig() (Config, error) {
	logger := logp.NewLogger(ModuleName)

	config := DefaultConfig()
	err := viper.UnmarshalKey("manager", &config)

	if err != nil {
		logger.Errorf("initConfig: Unmarshal failed with %s", err)
	}

	if config.MaxWorker <= 0 {
		logger.Panic("initConfig: MaxWorker should be larger than 0")
	}

	if config.WebPort <= 0 {
		logger.Panic("initConfig: WebPort should be larger than 0")
	}

	if config.ReqTimeout <= 100 {
		logger.Panic("initConfig: ReqTimeout should be larger than 100")
	}

	return config, nil
}

//to get current log level
func (m Manager) onGetLogLevel(c *gin.Context) {
	logger := m.logger.Named("web")

	for name, values := range c.Request.Header {
		logger.Infof("Http Header: %s : %s", name, values)
	}

	level := logp.GetLevel()
	c.JSON(200, GetLogLevelResponse{
		BaseResponse: BaseResponse{
			ErrCode: ErrCodeOk,
			ErrInfo: ErrInfoOk,
		},
		Level: level})
}

//to set log level
func (m Manager) onSetLogLevel(c *gin.Context) {
	logger := m.logger.Named("web")

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrCode: ErrCodeFailedToReadBody,
			ErrInfo: ErrInfoFailedToReadBody})
	}

	logger.Infof("onSetLevel: recv req: %s", string(body))

	var setLevelRequest SetLogLevelRequest
	err = json.Unmarshal(body, &setLevelRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrCode: ErrCodeFailedToParseBody,
			ErrInfo: ErrInfoFailedToParseBody})
		return
	}

	err = logp.SetLevel(setLevelRequest.Level)
	var rsp BaseResponse

	if err != nil {
		rsp = BaseResponse{ErrCode: ErrCodeGeneral,
			ErrInfo: err.Error()}
		c.JSON(http.StatusBadRequest, rsp)
	} else {
		rsp = BaseResponse{ErrCode: ErrCodeOk, ErrInfo: ErrInfoOk}
		c.JSON(200, rsp)
	}

	logger.Infof("onSetLevel: send rsp: %+v", rsp)
}