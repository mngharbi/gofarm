/*
	API used to interact with the server
*/

package gofarm

import (
	"errors"
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

func ResetServer() {
	serverSingleton.stateLock.Lock()
	serverSingleton = server{
		stateLock: serverSingleton.stateLock,
	}
	serverSingleton.stateLock.Unlock()
}

func InitServer(externalServer Server) error {
	serverSingleton.stateLock.Lock()

	if serverSingleton.isInitialized {
		serverSingleton.stateLock.Unlock()
		return errors.New(alreadyInitializedError)
	}

	serverSingleton.isFirstStart = true

	// Save external server functions
	serverSingleton.externalStart = func(config Config, firstStart bool) error {
		return externalServer.Start(config, firstStart)
	}
	serverSingleton.externalShutdown = func() error {
		return externalServer.Shutdown()
	}
	serverSingleton.externalWork = func(rq *Request) *Response {
		return externalServer.Work(rq)
	}

	serverSingleton.isInitialized = true

	serverSingleton.stateLock.Unlock()
	return nil
}

func StartServer(conf Config) (err error) {
	serverSingleton.stateLock.Lock()
	err = serverSingleton.start(&conf)
	serverSingleton.stateLock.Unlock()
	return
}

func ShutdownServer() (err error) {
	serverSingleton.stateLock.Lock()
	err = serverSingleton.shutdown(false)
	serverSingleton.stateLock.Unlock()
	return
}

func ForceShutdownServer() (err error) {
	serverSingleton.stateLock.Lock()
	err = serverSingleton.shutdown(true)
	serverSingleton.stateLock.Unlock()
	return
}

func MakeRequest(request Request) (chan *Response, error) {
	serverSingleton.stateLock.RLock()

	// Error out if server is not running
	if !serverSingleton.isRunning {
		serverSingleton.stateLock.RUnlock()
		return nil, errors.New(notRunningError)
	}

	// Build corresponding job and push it into server's incoming job pipe
	var reqJob job
	reqJob.responseChannel = make(chan *Response, 1)
	requestCopy := request
	reqJob.requestPtr = &requestCopy
	serverSingleton.jobPipe <- &reqJob

	serverSingleton.stateLock.RUnlock()

	return reqJob.responseChannel, nil
}
