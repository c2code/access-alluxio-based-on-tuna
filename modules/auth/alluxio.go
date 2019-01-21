package auth

import (
	"github.com/pkg/errors"
	"github.com/gin-gonic/gin"
	"encoding/json"
	"fmt"
	"hexmeet.com/haishen/tuna/utils"
	"net/http"
	"time"
	"github.com/Alluxio/alluxio-go/option"
	"io/ioutil"
	"strings"
)
/*********************Role-Based Access Control of Tenants****************************/

type AlluxioWebRequest struct {
	GUID      string       `json:"guid"`        //*
	RbactBaseRequest
	FileName  string       `json:"file_name"`
	NewName   string       `json:"new_name"`
	FileID    int          `json:"file_id"`    //the file handle
	Body      string       `json:"content"`
	Size      string       `json:"size"`        //default is 1G , xxM or xxG or xxT
	ClientIP  string
}

type AlluxioWebResponse struct {
	BaseResponse
	GUID      string       `json:"guid"`
	FileID    int          `json:"file_id"`    //the file handle
	Body      string       `json:"content"`    //files content
}

func (r *AlluxioWebRequest) webRequestParamCheck() error {
	if r.GUID == "" {
		return errors.Errorf("GUID not set")
	}

	if r.User == "" {
		return errors.Errorf("User not set")
	}

	if r.Domain == "" {
		return errors.Errorf("Domain not set")
	}

	return nil
}

/***************************1. send request to worker***********************************/
func (m Manager) alluxioRestCall(c *gin.Context) {
	logger := m.logger.Named("alluxio")

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrCode: ErrCodeFailedToReadBody,
			ErrInfo: ErrInfoFailedToReadBody})
	}

	logger.Infof(  "alluxio : recv req: %s, from client %s", string(body), c.ClientIP())

	var inReq AlluxioWebRequest
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

	var requestType string

	switch c.Request.URL.Path {
	case "/create-user" :
		requestType = RequestAlluxioCreateUser
	case "/delete-user" :
		requestType = RequestAlluxioDeleteUser
	case "/auth/open-file" :
		requestType = RequestAlluxioOpenFile
	case "/auth/read-content" :
		requestType = RequestAlluxioReadContent
	case "/auth/create-file" :
		requestType = RequestAlluxioCreateFile
	case "/auth/write-content" :
		requestType = RequestAlluxioWriteContent
	case "/auth/close-file" :
		requestType = RequestAlluxioCloseFile
	case "/auth/delete-file" :
		requestType = RequestAlluxioDeleteFile
	case "/auth/rename-file" :
		requestType = RequestAlluxioRenameFile
	default:
		c.JSON(http.StatusBadRequest, BaseResponse{ErrCode: ErrCodeFailedToParseBody,
			ErrInfo: ErrInfoFailedToParseBody,
			MoreInfo: fmt.Sprintf("preprocess err: %s", c.Request.URL.Path)})
		return
	}

	workerReq := WorkerRequest{Type: requestType,
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

	lrsp := genericRsp.(AlluxioWebResponse)

	jsonRsp, _ := json.Marshal(lrsp)
	logger.Infof("To send rsp: %s", string(jsonRsp))
	c.Data(http.StatusOK, ContentTypeJSON, jsonRsp)
}


/****************worker handle request,it will be called by Entry of worker*****************************/

func (m Manager)alluxioWorkerHandle (workerCtx *WorkerContext) {
	logger := workerCtx.logger

	var rsp AlluxioWebResponse
	baseResp := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
		MoreInfo: "",
	}

	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	fileID    := -1
	body      := ""

	switch workerCtx.workerRequest.Type {
	case RequestAlluxioCreateUser :
		logger.Infof("Guid:%s, begin to handle create usr info", workerCtx.workerRequest.GUID)

		err := m.alluxioCreateUser(workerCtx)

		if err != nil {
			baseResp.ErrCode = ErrCodeCreateUserFail
			baseResp.ErrInfo = ErrInfoCreateUserFail
			baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
		}

	case RequestAlluxioDeleteUser :
		logger.Infof("Guid:%s, begin to handle create usr info", workerCtx.workerRequest.GUID)

		err := m.alluxioDeleteUser(workerCtx)

		if err != nil {
			baseResp.ErrCode = ErrCodeDeleteUserFail
			baseResp.ErrInfo = ErrInfoDeleteUserFail
			baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
		}

	case RequestAlluxioOpenFile   :
		logger.Infof("Guid:%s, begin to handle open file", workerCtx.workerRequest.GUID)

		fileID, baseResp = m.alluxioOpenFile(workerCtx)

	case RequestAlluxioReadContent :
		logger.Infof("Guid:%s, begin to handle read content", workerCtx.workerRequest.GUID)

		body, baseResp = m.alluxioReadContent(workerCtx)

	case RequestAlluxioCreateFile  :
		logger.Infof("Guid:%s, begin to handle create file", workerCtx.workerRequest.GUID)

		fileID, baseResp = m.alluxioCreateFile(workerCtx)

	case RequestAlluxioWriteContent   :
		logger.Infof("Guid:%s, begin to handle write content", workerCtx.workerRequest.GUID)

		baseResp = m.alluxioWriteContent(workerCtx)

	case RequestAlluxioCloseFile  :
		logger.Infof("Guid:%s, begin to handle close file", workerCtx.workerRequest.GUID)

		baseResp = m.alluxioCloseFile(workerCtx)

	case RequestAlluxioDeleteFile :
		logger.Infof("Guid:%s, begin to handle delete file", workerCtx.workerRequest.GUID)

		baseResp = m.alluxioDeleteFile(workerCtx)


	case RequestAlluxioRenameFile :
		logger.Infof("Guid:%s, begin to handle rename file", workerCtx.workerRequest.GUID)

		baseResp = m.alluxioRenameFile(workerCtx)

	default:
		baseResp.ErrCode = ErrCodeGeneral
		baseResp.ErrInfo = "the Method is not matched"
	}

	rsp = AlluxioWebResponse {
		BaseResponse: baseResp,
		GUID  : webRequst.GUID,
		FileID: fileID,
		Body  : body,
	}

	m.workerSendRsp(workerCtx, rsp)
}

/*****************************Alluxio function*********************************************/
func (m Manager) alluxioCreateUser (workerCtx *WorkerContext) error {
	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger := workerCtx.logger
	user := webRequst.User
	domain := webRequst.Domain
	object := ""

	logger.Infof("User:%s, domain:%s will be created", user, domain)


	if user == domain {
		object = "/" + domain + "/"
	} else {
		object = "/" + domain + "/" + user + "/"
	}

	/*err := m.fs.CreateDirectory(object, &option.CreateDirectory{})

	if err != nil {
		logger.Infof("User:%s, domain:%s was created fail", user, domain)
		return err
	}*/

	m.rbactInsertPolicy(user, user, domain, object + "*", "*")

	return nil
}

func (m Manager) alluxioDeleteUser (workerCtx *WorkerContext) error {
	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger := workerCtx.logger
	user := webRequst.User
	domain := webRequst.Domain
	object := ""


	logger.Infof("User:%s, domain:%s will be removed", user, domain)

	if user == domain {
		object = "/" + domain + "/"
	} else {
		object = "/" + domain + "/" + user + "/"
	}

	/*err := m.fs.Delete(object, &option.Delete{})

	if err != nil {
		logger.Infof("User:%s, domain:%s was deleted fail", user, domain)
		return err
	}*/

	m.rbactDeletePolicy(user, user, domain, object + "*", "*")

	return nil
}

func (m Manager) alluxioOpenFile (workerCtx *WorkerContext) (int, BaseResponse) {
	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger := workerCtx.logger
	user   := webRequst.User
	domain := webRequst.Domain
	object := "/" + domain + "/" + user + "/" + webRequst.FileName
	baseResp := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
	}

	logger.Infof("User:%s, domain:%s will open %s", user, domain, object)

	if m.rbactCheckRights(user, domain, object, "read") {
		logger.Infof("User:%s, domain:%s was permitted to open %s", user, domain, object)
	} else {
		logger.Infof("User:%s, domain:%s was denied to open %s", user, domain, object)
		baseResp.ErrCode = ErrCodeUserDeny
		baseResp.MoreInfo = ErrInfoUserDeny
		return -1, baseResp
	}

	id, err := m.fs.OpenFile(object, &option.OpenFile{})

	if err != nil {
		baseResp.ErrCode = ErrCodeOpenFail
		baseResp.ErrInfo = ErrInfoOpenFail
		baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
		return -1, baseResp
	}

	return id, baseResp
}

func (m Manager) alluxioReadContent (workerCtx *WorkerContext) (string, BaseResponse) {

	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger    := workerCtx.logger
	user      := webRequst.User
	domain    := webRequst.Domain
	fileID    := webRequst.FileID
	object    := "/" + domain + "/" + user + "/" + webRequst.FileName
	baseResp := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
	}

	logger.Infof("User:%s, domain:%s will read %s", user, domain, object)

	if m.rbactCheckRights(user, domain, object, "read") {
		logger.Infof("User:%s, domain:%s was permitted to read %s", user, domain, object)
	} else {
		logger.Infof("User:%s, domain:%s was denied to read %s", user, domain, object)
		baseResp.ErrCode = ErrCodeUserDeny
		baseResp.MoreInfo = ErrInfoUserDeny
		return "", baseResp
	}

	r, err := m.fs.Read(fileID)

	if err != nil {
		baseResp.ErrCode = ErrCodeReadFail
		baseResp.ErrInfo = ErrInfoReadFail
		baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
		return "", baseResp
	}
	defer r.Close()

	content, err := ioutil.ReadAll(r)

	if err != nil {
		baseResp.ErrCode = ErrCodeReadFail
		baseResp.ErrInfo = ErrInfoReadFail
		baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
		return "", baseResp
	}
	return string(content), baseResp
}

func (m Manager) alluxioCreateFile (workerCtx *WorkerContext) (int,BaseResponse) {
	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger    := workerCtx.logger
	user      := webRequst.User
	domain    := webRequst.Domain
	object    := "/" + domain + "/" + user + "/" + webRequst.FileName
	baseResp := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
	}

	logger.Infof("User:%s, domain:%s will create %s", user, domain, object)

	if m.rbactCheckRights(user, domain, object, "write") {
		logger.Infof("User:%s, domain:%s was permitted to create %s", user, domain, object)
	} else {
		logger.Infof("User:%s, domain:%s was denied to create %s", user, domain, object)
		baseResp.ErrCode = ErrCodeUserDeny
		baseResp.MoreInfo = ErrInfoUserDeny
		return -1, baseResp
	}

	id, err := m.fs.CreateFile(object, &option.CreateFile{})
	if err != nil {
		baseResp.ErrCode = ErrCodeCreateFileFail
		baseResp.ErrInfo = ErrInfoCreateFileFail
		baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
		return -1, baseResp
	}
	return id, baseResp
}

func (m Manager) alluxioWriteContent (workerCtx *WorkerContext) BaseResponse {
	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger    := workerCtx.logger
	user      := webRequst.User
	domain    := webRequst.Domain
	fileID    := webRequst.FileID
	object    := "/" + domain + "/" + user + "/" + webRequst.FileName
	baseResp  := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
	}

	logger.Infof("User:%s, domain:%s will write %s", user, domain, object)

	if m.rbactCheckRights(user, domain, object, "write") {
		logger.Infof("User:%s, domain:%s was permitted to write %s", user, domain, object)
	} else {
		logger.Infof("User:%s, domain:%s was denied to write %s", user, domain, object)
		baseResp.ErrCode = ErrCodeUserDeny
		baseResp.MoreInfo = ErrInfoUserDeny
		return baseResp
	}

	_, err := m.fs.Write(fileID, strings.NewReader(webRequst.Body))

	if err != nil {
		baseResp.ErrCode = ErrCodeWriteFail
		baseResp.ErrInfo = ErrInfoWriteFail
		baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
	}

	return baseResp
}

func (m Manager) alluxioCloseFile (workerCtx *WorkerContext) BaseResponse {

	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger    := workerCtx.logger
	user      := webRequst.User
	domain    := webRequst.Domain
	fileID    := webRequst.FileID
	object    := "/" + domain + "/" + user + "/" + webRequst.FileName

	baseResp  := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
	}

	logger.Infof("User:%s, domain:%s will close %s", user, domain, object)

	if m.rbactCheckRights(user, domain, object, "read") || m.rbactCheckRights(user, domain, object, "write") {
		logger.Infof("User:%s, domain:%s was permitted to close %s", user, domain, object)
	} else {
		logger.Infof("User:%s, domain:%s was denied to close %s", user, domain, object)
		baseResp.ErrCode = ErrCodeUserDeny
		baseResp.MoreInfo = ErrInfoUserDeny
		return baseResp
	}

	m.fs.Close(fileID)

	return baseResp
}

func (m Manager) alluxioDeleteFile (workerCtx *WorkerContext) BaseResponse {

	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger    := workerCtx.logger
	user      := webRequst.User
	domain    := webRequst.Domain
	object    := "/" + domain + "/" + user + "/" + webRequst.FileName

	baseResp  := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
	}

	logger.Infof("User:%s, domain:%s will delete %s", user, domain, object)

	if m.rbactCheckRights(user, domain, object, "write") {
		logger.Infof("User:%s, domain:%s was permitted to delete %s", user, domain, object)
	} else {
		logger.Infof("User:%s, domain:%s was denied to delete %s", user, domain, object)
		baseResp.ErrCode = ErrCodeUserDeny
		baseResp.MoreInfo = ErrInfoUserDeny
		return baseResp
	}

	err := m.fs.Delete(object, &option.Delete{})

	if err != nil {
		baseResp.ErrCode = ErrCodeDeleteFileFail
		baseResp.ErrInfo = ErrInfoDeleteFileFail
		baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
	}

	return baseResp
}

func (m Manager) alluxioRenameFile (workerCtx *WorkerContext) BaseResponse {

	webRequst := workerCtx.workerRequest.Body.(AlluxioWebRequest)
	logger    := workerCtx.logger
	user      := webRequst.User
	domain    := webRequst.Domain
	object    := "/" + domain + "/" + user + "/" + webRequst.FileName
	newName   := "/" + domain + "/" + user + "/" + webRequst.NewName

	baseResp  := BaseResponse {
		ErrCode: ErrCodeOk,
		ErrInfo: ErrInfoOk,
	}

	logger.Infof("User:%s, domain:%s will rename %s", user, domain, object)

	if m.rbactCheckRights(user, domain, object, "write") {
		logger.Infof("User:%s, domain:%s was rename to delete %s", user, domain, object)
	} else {
		logger.Infof("User:%s, domain:%s was rename to delete %s", user, domain, object)
		baseResp.ErrCode = ErrCodeUserDeny
		baseResp.MoreInfo = ErrInfoUserDeny
		return baseResp
	}

	err := m.fs.Rename(object, newName, &option.Rename{})

	if err != nil {
		baseResp.ErrCode = ErrCodeRenameFileFail
		baseResp.ErrInfo = ErrInfoRenameFileFail
		baseResp.MoreInfo = fmt.Sprintf("Err: %s", err)
	}

	return baseResp
}
