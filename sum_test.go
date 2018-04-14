/*
	Testing with a simple, concurrent, integer sum server
*/

package gofarm

import (
	"errors"
	"math/rand"
	"sync"
	"testing"
)

/*
	Global vars
*/

const functionalTestingNumWorkers = 4
const testFibonnaciMaxNum = 1000000

/*
	Required definitions
*/

type sumServer struct {
	lock         sync.RWMutex
	sum          int
	startFail    bool
	shutdownFail bool
}

type sumRequest struct {
	isRead bool
	delta  int
}

type sumResponse struct {
	sum int
}

func (sv *sumServer) Start(_ Config, isFirstTime bool) (err error) {
	if isFirstTime {
		sv.sum = 0
	}
	err = nil
	if sv.startFail {
		err = errors.New("")
	}
	return
}

func (sv *sumServer) Shutdown() error {
	if sv.shutdownFail {
		return errors.New("")
	}
	return nil
}

func (sv *sumServer) Work(rqPtrNative *Request) *Response {
	rqPtr := (*rqPtrNative).(*sumRequest)
	respPtr := &sumResponse{}
	if rqPtr.isRead {
		sv.lock.RLock()
		respPtr.sum = sv.sum
		sv.lock.RUnlock()
	} else {
		sv.lock.Lock()
		sv.sum += rqPtr.delta
		respPtr.sum = calculateRandomFib()
		sv.lock.Unlock()
	}
	var respNative Response
	respNative = respPtr
	return &respNative
}

/*
	Test helpers
*/

func makeRandomRequest(sh *ServerHandler) (err error) {
	_, err = sh.MakeRequest(&sumRequest{
		isRead: false,
		delta:  rand.Intn(100) + 1,
	})
	return
}

func calculateRandomFib() int {
	n := rand.Intn(testFibonnaciMaxNum) + 1
	first := 0
	second := 1
	for i := 0; i < n; i++ {
		first, second = second, first+second
	}
	return first
}

func getConfig(workers int) Config {
	return Config{
		NumWorkers: workers,
	}
}

func resetContext(startFail bool, shutdownFail bool) (sh *ServerHandler, err error) {
	sh = ProvisionServer()
	err = sh.InitServer(&sumServer{
		startFail:    startFail,
		shutdownFail: shutdownFail,
	})
	return
}

/*
	Server setup testing
*/
func TestStartShutdown(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	sh.StartServer(getConfig(functionalTestingNumWorkers))
	sh.ShutdownServer()
}

func TestStartExternalError(t *testing.T) {
	sh, initErr := resetContext(true, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	startError := sh.StartServer(getConfig(functionalTestingNumWorkers))
	if startError == nil {
		t.Errorf("Start should fail if external shutdown fails.")
	}
}

func TestShutdownFirst(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	shutdownErr := sh.ShutdownServer()
	if shutdownErr == nil {
		t.Errorf("Shutdown before start should fail.")
	}
}

func TestShutdownExternalError(t *testing.T) {
	sh, initErr := resetContext(false, true)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	sh.StartServer(getConfig(functionalTestingNumWorkers))

	shutdownError := sh.ShutdownServer()
	if shutdownError == nil {
		t.Errorf("Shutdown should fail if external shutdown fails.")
	}
}

func TestRequestFirst(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	err := makeRandomRequest(sh)
	if err == nil {
		t.Errorf("Request while server is down should fail.")
	}
}

func TestDoubleStart(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	firstStartErr := sh.StartServer(getConfig(functionalTestingNumWorkers))
	if firstStartErr != nil {
		t.Errorf("Starting should work.")
	}

	secondStartErr := sh.StartServer(getConfig(functionalTestingNumWorkers))
	if secondStartErr == nil {
		t.Errorf("Starting twice should fail.")
	}

	shutdownErr := sh.ShutdownServer()
	if shutdownErr != nil {
		t.Errorf("Shutdown after double start should work.")
	}
}

func TestDoubleInit(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	doubleInitErr := sh.InitServer(&sumServer{})
	if doubleInitErr == nil {
		t.Errorf("Calling init twice should fail.")
	}
}

/*
	Decommission testing
*/
func testDecommision(t *testing.T, startServer bool, forceDecommision bool, shutdownFailure bool) {
	sh, initErr := resetContext(false, shutdownFailure)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	runningText := "not"
	if startServer {
		runningText = ""
		err := sh.StartServer(getConfig(functionalTestingNumWorkers))
		if err != nil {
			t.Errorf("Starting server should not fail.")
		}
	}

	var err error
	forceText := ""
	if forceDecommision {
		forceText = "Force"
		err = sh.ForceDecommissionServer()
	} else {
		err = sh.DecommissionServer()
	}

	if shutdownFailure && err == nil {
		t.Errorf("%v Decommision should fail while server is %v running and external shutdown fails.", forceText, runningText)
	} else if !shutdownFailure && err != nil {
		t.Errorf("%v Decommision should not fail while server is %v running.", forceText, runningText)
	}
}

func TestProvisionDecommision(t *testing.T) {
	testDecommision(t, false, false, false)
}

func TestProvisionDecommisionRunning(t *testing.T) {
	testDecommision(t, true, false, false)
}

func TestForceProvisionDecommision(t *testing.T) {
	testDecommision(t, false, true, false)
}

func TestForceProvisionDecommisionRunning(t *testing.T) {
	testDecommision(t, true, true, false)
}

func TestForceProvisionDecommisionRunningExternalFail(t *testing.T) {
	testDecommision(t, true, true, true)
}

/*
	Functional testing
*/
func TestOneAdd(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	// Make request and shutdown server
	expected := 10
	sh.StartServer(getConfig(functionalTestingNumWorkers))
	sh.MakeRequest(&sumRequest{
		isRead: false,
		delta:  expected,
	})
	sh.ShutdownServer()

	// Restart server and get result
	sh.StartServer(getConfig(functionalTestingNumWorkers))
	responseChan, err := sh.MakeRequest(&sumRequest{
		isRead: true,
	})
	responseNativePtr := <-responseChan
	resp := (*responseNativePtr).(*sumResponse)
	sh.ShutdownServer()

	if err != nil || resp.sum != expected {
		t.Errorf("One sum request failed. results:\n result: %v\n expected: %v\n", resp.sum, expected)
	}
}

func TestManyAddsOneStart(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	// Make requests and shutdown server
	testCases := 10000
	maxNum := 100
	randArray := make([]int, testCases)
	channelArray := make([](chan *Response), testCases)
	sh.StartServer(getConfig(functionalTestingNumWorkers))
	for i := 0; i < testCases; i++ {
		randArray[i] = rand.Intn(maxNum) + 1
		channelArray[i], _ = sh.MakeRequest(&sumRequest{
			isRead: false,
			delta:  randArray[i],
		})
	}
	sh.ShutdownServer()

	// Count expected based on successful requests
	expected := 0
	for requestIndex, requestChannel := range channelArray {
		_, ok := <-requestChannel
		if ok {
			expected += randArray[requestIndex]
		} else {
			t.Errorf("A request was interrupted during soft shutdown")
		}
	}

	// Restart server and get result
	sh.StartServer(getConfig(functionalTestingNumWorkers))
	responseChan, err := sh.MakeRequest(&sumRequest{
		isRead: true,
	})
	responseNativePtr := <-responseChan
	resp := (*responseNativePtr).(*sumResponse)
	sh.ShutdownServer()

	if err != nil || resp.sum != expected {
		t.Errorf("Soft shutdown: multiple sum requests failed. results:\n result: %v\n expected: %v\n", resp.sum, expected)
	}
}

func TestForceShutdownManyAddsOneStart(t *testing.T) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		t.Errorf(initErr.Error())
		return
	}

	// Make requests and shutdown server
	testCases := 10000
	maxNum := 100
	randArray := make([]int, testCases)
	channelArray := make([](chan *Response), testCases)
	sh.StartServer(getConfig(functionalTestingNumWorkers))
	for i := 0; i < testCases; i++ {
		randArray[i] = rand.Intn(maxNum) + 1
		channelArray[i], _ = sh.MakeRequest(&sumRequest{
			isRead: false,
			delta:  randArray[i],
		})
	}
	sh.ForceShutdownServer()

	// Count expected based on successful requests
	expected := 0
	for requestIndex, requestChannel := range channelArray {
		_, ok := <-requestChannel
		if ok {
			expected += randArray[requestIndex]
		}
	}

	// Restart server and get result
	sh.StartServer(getConfig(functionalTestingNumWorkers))
	responseChan, err := sh.MakeRequest(&sumRequest{
		isRead: true,
	})
	responseNativePtr := <-responseChan
	resp := (*responseNativePtr).(*sumResponse)
	sh.ShutdownServer()

	if err != nil || resp.sum != expected {
		t.Errorf("Force shutdown, multiple sum requests failed. results:\n result: %v\n expected: %v\n", resp.sum, expected)
	}
}

/*
	Benchmarking server performance
*/

func BenchmarkWrite1Worker(b *testing.B) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		b.Errorf(initErr.Error())
		return
	}

	sh.StartServer(getConfig(1))
	for n := 0; n < b.N; n++ {
		makeRandomRequest(sh)
	}
	sh.ShutdownServer()
}

func BenchmarkWrite2Worker(b *testing.B) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		b.Errorf(initErr.Error())
		return
	}

	sh.StartServer(getConfig(2))
	for n := 0; n < b.N; n++ {
		makeRandomRequest(sh)
	}
	sh.ShutdownServer()
}

func BenchmarkWrite4Worker(b *testing.B) {
	sh, initErr := resetContext(false, false)
	if initErr != nil {
		b.Errorf(initErr.Error())
		return
	}

	sh.StartServer(getConfig(4))
	for n := 0; n < b.N; n++ {
		makeRandomRequest(sh)
	}
	sh.ShutdownServer()
}
