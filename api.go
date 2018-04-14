/*
	API used to interact with the server
*/

package gofarm

import (
	"errors"
	"sync"
)

/*
	Error messages
*/

const notRunningError = "Server not running."
const alreadyInitializedError = "Server already initialized."
const shuttingDownError = "Server shutting down."

/*
	API
*/

func ProvisionServer() (sh *ServerHandler) {
	rackLock.Lock()
	serverNb := len(serverRack)
	serverPtr := &server{
		stateLock: &sync.RWMutex{},
	}
	sh = &ServerHandler{
		internalIndex: serverNb,
		serverPtr:     serverPtr,
	}
	serverRack = append(serverRack, sh)
	rackLock.Unlock()
	return
}

func (sh *ServerHandler) doDecommission(force bool) (err error) {
	rackLock.Lock()
	sh.serverPtr.stateLock.Lock()
	err = nil
	if sh.serverPtr.isRunning {
		err = sh.serverPtr.shutdown(force)
	}
	if err == nil {
		serverRack = append(serverRack[:sh.internalIndex], serverRack[sh.internalIndex+1:]...)
	}
	sh.serverPtr.stateLock.Unlock()
	rackLock.Unlock()
	return
}

func (sh *ServerHandler) ForceDecommissionServer() error {
	return sh.doDecommission(true)
}

func (sh *ServerHandler) DecommissionServer() error {
	return sh.doDecommission(false)
}

func (sh *ServerHandler) ResetServer() {
	sh.serverPtr.stateLock.Lock()
	sh.serverPtr = &server{
		stateLock: sh.serverPtr.stateLock,
	}
	sh.serverPtr.stateLock.Unlock()
}

func (sh *ServerHandler) InitServer(externalServer Server) error {
	sh.serverPtr.stateLock.Lock()

	if sh.serverPtr.isInitialized {
		sh.serverPtr.stateLock.Unlock()
		return errors.New(alreadyInitializedError)
	}

	sh.serverPtr.isFirstStart = true

	// Save external server functions
	sh.serverPtr.externalStart = func(config Config, firstStart bool) error {
		return externalServer.Start(config, firstStart)
	}
	sh.serverPtr.externalShutdown = func() error {
		return externalServer.Shutdown()
	}
	sh.serverPtr.externalWork = func(rq *Request) *Response {
		return externalServer.Work(rq)
	}

	sh.serverPtr.isInitialized = true

	sh.serverPtr.stateLock.Unlock()
	return nil
}

func (sh *ServerHandler) StartServer(conf Config) (err error) {
	sh.serverPtr.stateLock.Lock()
	err = sh.serverPtr.start(&conf)
	sh.serverPtr.stateLock.Unlock()
	return
}

func (sh *ServerHandler) ShutdownServer() (err error) {
	sh.serverPtr.stateLock.Lock()
	err = sh.serverPtr.shutdown(false)
	sh.serverPtr.stateLock.Unlock()
	return
}

func (sh *ServerHandler) ForceShutdownServer() (err error) {
	sh.serverPtr.stateLock.Lock()
	err = sh.serverPtr.shutdown(true)
	sh.serverPtr.stateLock.Unlock()
	return
}

func (sh *ServerHandler) MakeRequest(request Request) (chan *Response, error) {
	sh.serverPtr.stateLock.RLock()

	// Error out if server is not running
	if !sh.serverPtr.isRunning {
		sh.serverPtr.stateLock.RUnlock()
		return nil, errors.New(notRunningError)
	}

	// Build corresponding job and push it into server's incoming job pipe
	var reqJob job
	reqJob.responseChannel = make(chan *Response, 1)
	requestCopy := request
	reqJob.requestPtr = &requestCopy
	sh.serverPtr.jobPipe <- &reqJob

	sh.serverPtr.stateLock.RUnlock()

	return reqJob.responseChannel, nil
}
