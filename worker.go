/*
	Worker implementation
*/

package gofarm

/*
	Internal types
*/

type worker struct {
	index int
	jobs  chan *job
	sv    *server
}

/*
	Main internal functions: run/shutdown
*/

func (wk *worker) run() {
	for {
		// Accept incoming jobs
		newJob, ok := <-wk.jobs

		if ok {
			// Get response through custom Work function
			var responsePtr *Response
			responsePtr = wk.sv.externalWork(newJob.requestPtr)
			newJob.responseChannel <- responsePtr

			// Report to server that we're free
			wk.sv.workerPipe <- wk
		} else {
			break
		}
	}

	wk.sv.workerWaitGroup.Done()
}

func (wk *worker) shutdown() {
	close(wk.jobs)
}
