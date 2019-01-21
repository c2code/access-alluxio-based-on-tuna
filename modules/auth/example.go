package auth

import (
	"github.com/gin-gonic/gin"
	"encoding/json"
	"fmt"
	"net/http"
	"hexmeet.com/haishen/tuna/utils"
	"github.com/pkg/errors"
	"time"
)

type ExampleWebRequest struct {
	GUID      string       `json:"guid"`
	Test      string       `json:"test"`
	ClientIP  string
}


type ExampleWebResponse struct {
	BaseResponse
	GUID      string       `json:"guid"`
	Test      string       `json:"test"`
}

func (e *ExampleWebRequest) webRequestParamCheck() error {
	if e.GUID == "" {
		return errors.Errorf("GUID not set")
	}

	return nil
}

/***************************The func is a callback for rest api **************************************/

//the func is only example for post handle, add a request to master dispatch channel and wait result of worker handle

func (m Manager) exampleRestCall(c *gin.Context) {
	logger := m.logger.Named("example")

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrCode: ErrCodeFailedToReadBody,
			ErrInfo: ErrInfoFailedToReadBody})
	}

	logger.Infof("example: recv req: %s", string(body))

	var inReq ExampleWebRequest////////////need to modify
	err = json.Unmarshal(body, &inReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrCode: ErrCodeFailedToParseBody,
			ErrInfo: ErrInfoFailedToParseBody,
			MoreInfo: fmt.Sprintf("Unmarshal err: %s", err)})
		return
	}

	err = inReq.webRequestParamCheck()
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrCode: ErrCodeFailedToParseBody,
			ErrInfo: ErrInfoFailedToParseBody,
			MoreInfo: fmt.Sprintf("preprocess err: %s", err)})
		return
	}

	inReq.ClientIP = c.ClientIP()

	guid := utils.NewUUID()
	rspChan := make(chan interface{})
	doneChan := make(chan bool)
	timeoutChan := time.After(time.Duration(m.config.ReqTimeout) * time.Millisecond)

	workerReq := WorkerRequest{Type: RequestExample,
		GUID:     guid,
		Body:     inReq,
		RspChan:  rspChan,
		DoneChan: doneChan}

	select {
	case m.dispatchChan <- workerReq:
		break
	case <-timeoutChan:
		logger.Errorf("Failed to send request %+v to dispatcher, timeout",
			inReq)
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrCode: ErrCodeTimeout,
			ErrInfo: ErrInfoTimeout})
		return
	}

	var genericRsp interface{}

	select {
	case genericRsp = <-rspChan:
		break
	case <-timeoutChan:
		logger.Errorf("Failed to recv response %+v from worker, timeout",
			inReq)
		close(doneChan)
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrCode: ErrCodeTimeout,
			ErrInfo: ErrInfoTimeout})
		return
	}

	lrsp := genericRsp.(ExampleWebResponse) ////////////need to modify

	jsonRsp, _ := json.Marshal(lrsp)
	logger.Infof("To send rsp: %s", string(jsonRsp))
	c.Data(http.StatusOK, ContentTypeJSON, jsonRsp)
}

/************************The func will be called by worker entry by task type*******************************/

//handle the request of master, it will be call by work entry
func (m Manager)exampleWorkerHandle (workerCtx *WorkerContext) {
	//logger := workerCtx.logger

	var rsp ExampleWebResponse////////////need to modify

	webRequst := workerCtx.workerRequest.Body.(ExampleWebRequest)////////////need to modify

	rsp = ExampleWebResponse {////////////need to modify
		BaseResponse: BaseResponse{
			ErrCode: ErrCodeOk,
			ErrInfo: ErrInfoOk,
			MoreInfo: "I am only a example!",
		},
		GUID: webRequst.GUID,
		Test: "I am only a example!",////////////need to modify
	}

	m.workerSendRsp(workerCtx, rsp)
}
