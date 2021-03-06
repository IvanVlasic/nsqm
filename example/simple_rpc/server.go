// +build server

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/minus5/nsqm"
	"github.com/minus5/nsqm/rpc"
)

const (
	reqTopic = "request" // topic for listening to requests
	channel  = "server"  // channel name on that topic
)

func main() {
	// configuration
	cfgr := nsqm.Local()

	// nsq producer for sending responses
	producer, err := nsqm.NewProducer(cfgr)
	if err != nil {
		log.Fatal(err)
	}

	// rpc server: accepts request, calls application, sends response
	ctx, cancel := context.WithCancel(context.Background())
	appServer := &server{}
	rpcServer := rpc.NewServer(ctx, appServer, producer)

	// consumer arround rpcServer
	consumer, err := nsqm.NewConsumer(cfgr, reqTopic, channel, rpcServer)
	if err != nil {
		log.Fatal(err)
	}

	// clean exit
	defer producer.Stop() // 3. stop response producer
	defer cancel()        // 2. cancel any pending operation (returns unfinished messages to nsq)
	defer consumer.Stop() // 1. stop accepting new requests

	waitForInterupt()
}

func waitForInterupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}

type server struct{}

// Server is entry point for all rpc requests
// method is the name of the server method
// reqBuf request data
// returns:
//   response data
//   application error
func (s *server) Serve(ctx context.Context, method string, reqBuf []byte) ([]byte, error) {
	switch method {
	case "Add":
		// unpack
		var req request
		err := json.Unmarshal(reqBuf, &req)
		if err != nil {
			return nil, err
		}
		// call actual server method
		z := s.add(req.X, req.Y)
		// pack
		rsp := response{Z: z}
		rspBuf, err := json.Marshal(rsp)
		if err != nil {
			return nil, err
		}
		return rspBuf, nil
	default:
		return nil, nil
	}
}

// actual server method
func (s *server) add(x, y int) int {
	return x + y
}

// dto structs

type request struct {
	X int
	Y int
}

type response struct {
	Z int
}
