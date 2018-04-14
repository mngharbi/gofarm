/*
	Exported types used in the API
*/

package gofarm

type Config struct {
	NumWorkers int
}

type Response interface{}

type Request interface{}

type Server interface {
	Start(config Config, firstStart bool) error
	Shutdown() error
	Work(*Request) *Response
}

/*
	Reference used to interact with servers
*/
type ServerHandler struct {
	internalIndex int
	serverPtr     *server
}
