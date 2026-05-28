package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	webPort  = "80"
	rpcPort  = "5001"
	mongoURL = "mongodb://mongo:27017"
	gRpcPort = "50001"
)

// mongo vs code connection string = mongodb://admin:password@localhost:27017/

var client *mongo.Client

type Config struct {
	Models data.Models
}

func main() {
	// connect to mongo
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panic(err)
	}
	client = mongoClient

	// create a context in order to disconnect
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// close connection
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	app := Config{
		Models: data.New(client),
	}

	// register the rpc server
	err = rpc.Register(new(RPCServer))
	go app.rpcListen()

	// start webserve
	log.Println("Starting Service on Port", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

// rpcListen starts an RPC server that listens for remote procedure calls
// It accepts connections and handles them asynchronously using Go's net/rpc package
func (app *Config) rpcListen() error {
	log.Println("Starting RPC Server on port : ", rpcPort)

	// Create a TCP listener on all interfaces (0.0.0.0) and the specified RPC port
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", rpcPort))
	if err != nil {
		return err
	}

	// Ensure the listener is closed when the function exits
	defer listen.Close()

	// Infinite loop to continuously accept incoming RPC connections
	for {
		// Accept a new connection from a client
		rpcConn, err := listen.Accept()
		if err != nil {
			// If there's an error accepting, skip this iteration and continue listening
			continue
		}

		// Handle the connection in a separate goroutine to allow concurrent requests
		// ServeConn blocks until the client disconnects
		go rpc.ServeConn(rpcConn)
	}
}

func connectToMongo() (*mongo.Client, error) {
	// create connection options
	clientOptions := options.Client().ApplyURI(mongoURL)
	clientOptions.SetAuth(options.Credential{
		Username: "admin",
		Password: "password",
	})

	// connect
	// c, err := mongo.Connect(context.TODO(), clientOptions)
	c, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Println("Error Connecting:", err)
		return nil, err
	}
	log.Println("Connected to Mongo")
	return c, nil
}
