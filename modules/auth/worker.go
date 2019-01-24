package auth

import (
	"hexmeet.com/haishen/tuna/logp"
)

// WorkerContext def
type WorkerContext struct {
	logger          *logp.Logger
	workerRequest   WorkerRequest
}

//Entry of worker
func (m Manager) work(workID string) {
	//create the worker channel to receive a master request
	workerChan := make(chan interface{})

	logger := m.logger.Named(workID)

	defer logger.Infof("%s is closed", workID)

	var inRequest interface{}

	workerCtx := &WorkerContext{logger: logger}

	for {
		m.freeWorkerChan <- workerChan //add the worker channel into free worker channel of master

		select {
		case <-m.doneChan: //when the master close, the worker will close
			return
		case inRequest = <-workerChan: //get the task from the worker channel itself
			break
		}

		switch tmp := inRequest.(type) { //check the request struct type if it is  WorkerRequest struct
		case WorkerRequest:

			workerCtx.workerRequest = tmp

			switch workerCtx.workerRequest.Type {
			//case RequestExample:
				//m.exampleWorkerHandle(workerCtx) //it has not been run, it is only a example
			case RequestAlluxioCreateUser,
				RequestAlluxioDeleteUser,
			    RequestAlluxioCreateFile,
				RequestAlluxioWriteContent,
			    RequestAlluxioOpenFile,
				RequestAlluxioReadContent,
			    RequestAlluxioDeleteFile,
			    RequestAlluxioRenameFile,
				RequestAlluxioUploadFile,
				RequestAlluxioReadFile:
				m.alluxioWorkerHandle(workerCtx)
			default:
				logger.Errorf("Unexpected worker request type: %s", workerCtx.workerRequest.Type)
			}
		default:
			logger.Error("Unexpected request type")
		}
	}
}

//send the result of worker to master by channel of worker request
func (m Manager) workerSendRsp(workerCtx *WorkerContext, rsp interface{}) {
	workerCtx.logger.Debugf("to send rsp for %s ,%+v", workerCtx.workerRequest.GUID, rsp)
	select {
	case <-m.doneChan:
		return
	case <-workerCtx.workerRequest.DoneChan:
		workerCtx.logger.Debug("requester doesn't want rsp for %s  %s",
			workerCtx.workerRequest.Type, workerCtx.workerRequest.GUID)
		break
	case workerCtx.workerRequest.RspChan <- rsp:
		break
	}
}

