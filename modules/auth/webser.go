package auth

import (
	"github.com/gin-gonic/gin"
	"fmt"
	"hexmeet.com/haishen/tuna/thirdparty/github.com/gin-contrib/cors"
	"hexmeet.com/haishen/tuna/thirdparty/github.com/gin-contrib/static"
	"hexmeet.com/haishen/tuna/utils"
)
type PingResponse struct {
	BaseResponse
	Message string `json:"message"`
}

//entry of web server
func (m Manager) webListen() {
	ginLogger := m.logger.Named("gin")

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(utils.Ginzap(ginLogger))
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	router.Use(static.Serve("/", static.LocalFile("./dist", true)))
	router.Use(static.Serve("/auth", static.LocalFile("./dist", true)))

	//provide a internal access rest api
	tuna_v1:= router.Group("/")
	{
		tuna_v1.POST("/create-user", m.alluxioRestCall)
		tuna_v1.POST("/delete-user", m.alluxioRestCall)
	}

	//provide a external access rest api
	tuna_v2 := router.Group("/auth")
	{
		tuna_v2.GET("/ping", m.onPing) //to check the tuna service is accessful
		tuna_v2.GET("/log-level", m.onGetLogLevel) //get log level
		tuna_v2.POST("/log-level", m.onSetLogLevel) //set log level

		tuna_v2.POST("/create-file", m.alluxioRestCall)
		tuna_v2.POST("/write-content", m.alluxioRestCall)
		tuna_v2.POST("/open-file", m.alluxioRestCall)
		tuna_v2.POST("/read-content", m.alluxioRestCall)
		tuna_v2.POST("/close-file", m.alluxioRestCall)
		tuna_v2.POST("/delete-file", m.alluxioRestCall)
		tuna_v2.POST("/rename-file", m.alluxioRestCall)
	}

	portSpec := fmt.Sprintf(":%d", m.config.WebPort)

	router.Run(portSpec)
}

func (m Manager) onPing(c *gin.Context) {
	c.JSON(200, PingResponse{
		BaseResponse: BaseResponse{
			ErrCode: ErrCodeOk,
			ErrInfo: ErrInfoOk,
		},
		Message: "pong"})
}

