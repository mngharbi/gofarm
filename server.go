/*
	Server implementation
*/

package gofarm

import (
	"errors"
	"sync"
)

/*
	Error messages
*/

const startWhileRunningError = "Can't start server while it's running."
const shutdownWhileNotRunningError = "Can't shutdown server while it's not running."

/*
	Global vars
*/
var (
	serverSingleton server = server{
		stateLock: &sync.RWMutex{},
	}
)

/*
	Internal types
*/

type server struct {
	isInitialized		bool
	isFirstStart		bool
	isRunning			bool
	isShuttingDown		bool
	isForceShuttingDown	bool

	workerPool		[]*worker
	freeWorkers		[]*worker
	jobPool			[]*job
	jobPipe			chan *job
	workerPipe		chan *worker
	shutdownPipe	chan bool
	workerWaitGroup	sync.WaitGroup

	// Used to control requests vs start/shutdown
	stateLock *sync.RWMutex

	// External custom functions defined on initialization
	externalStart func(config Config, firstStart bool) error
	externalShutdown func() error
	externalWork func(rq *Request) *Response
}

type job struct {
	responseChannel		chan *Response
	requestPtr			*Request
}

/*
	Main internal functions: start/run/shutdown
*/

func (sv *server) start (conf *Config) error {
	if sv.isRunning {
		return errors.New(startWhileRunningError)
	}

	// Call custom initialization
	customInitErr := sv.externalStart(*conf, sv.isFirstStart)
	if customInitErr != nil {
		return customInitErr
	}

	// Setup channels
	sv.jobPipe = make(chan *job)
	sv.workerPipe = make(chan *worker, conf.NumWorkers)
	sv.shutdownPipe = make(chan bool)

	// Build workers
	sv.workerPool = make([]*worker, conf.NumWorkers)
	sv.freeWorkers = make([]*worker, conf.NumWorkers)
	for workerIndex := 0; workerIndex < conf.NumWorkers ; workerIndex++ {
		newWorker := worker{
			jobs: make(chan *job),
			sv:	sv,
			index: workerIndex,
		}
		sv.workerPool[workerIndex] = &newWorker
		sv.freeWorkers[workerIndex] = &newWorker
	}

	// Startup workers
	sv.workerWaitGroup.Add(conf.NumWorkers)
	for _,worker := range sv.workerPool {
		go worker.run()
	}

	// Start running server
	go sv.run()

	// Set flags
	sv.isFirstStart = false
	sv.isRunning = true

	return nil
}

func (sv *server) run () {
	for {
		select {
		// Accept incoming jobs
		case newJob := <- sv.jobPipe:
			sv.jobPool = append(sv.jobPool, newJob)
			break

		// Accept status update from worker
		case freeWorker := <- sv.workerPipe:
			sv.freeWorkers = append(sv.freeWorkers, freeWorker)
			break

		// Shutdown
		case forceShutdown := <- sv.shutdownPipe:
			sv.isShuttingDown = true
			sv.isForceShuttingDown = forceShutdown
			break
		}

		// Distribute available jobs to avaiable workers in order
		if !sv.isForceShuttingDown {
			sv.distributeJobs()
		}

		// If force shutting down or work is done, signal workers to die
		if sv.isForceShuttingDown ||
			sv.isShuttingDown && len(sv.jobPool) == 0 && len(sv.freeWorkers) == len(sv.workerPool) {
			for _,worker := range sv.workerPool {
				go worker.shutdown()
			}

			// Close channels for pending jobs
			if sv.isForceShuttingDown {
				for _,job := range sv.jobPool {
					close(job.responseChannel)
				}
			}

			// Reset defaults
			sv.workerPool = []*worker{}
			sv.freeWorkers = []*worker{}
			sv.jobPool = []*job{}
			sv.isForceShuttingDown = false
			sv.isShuttingDown = false

			break
		}
	}
}

func (sv *server) shutdown (force bool) (err error) {
	if !sv.isRunning {
		return errors.New(shutdownWhileNotRunningError)
	}

	// Call custom shutdown function
	err = sv.externalShutdown()
	if err != nil {
		return
	}

	sv.isRunning = false

	// Signal to shutdown
	sv.shutdownPipe <- force

	// Wait for complete shutdown
	sv.workerWaitGroup.Wait()

	return
}


/*
	Helpers
*/

func (sv *server) distributeJobs () {
	// Count maximum amount of assignments possible
	jobPoolLen := len(sv.jobPool)
	freeWorkersLen := len(sv.freeWorkers)
	assignmentsCount := jobPoolLen
	if freeWorkersLen < jobPoolLen {
		assignmentsCount = freeWorkersLen
	}
	if assignmentsCount == 0 {
		return
	}

	// Remove jobs/workers to be assigned
	jobsToAssign := sv.jobPool[:assignmentsCount]
	workersToAssign := sv.freeWorkers[:assignmentsCount]
	sv.jobPool = sv.jobPool[assignmentsCount:]
	sv.freeWorkers = sv.freeWorkers[assignmentsCount:]

	// Put jobs into their worker's incoming channel
	for i := 0; i < assignmentsCount; i++ {
		workersToAssign[i].jobs <- jobsToAssign[i]
	}
}
