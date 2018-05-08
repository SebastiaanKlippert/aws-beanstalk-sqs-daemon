package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/SebastiaanKlippert/aws-beanstalk-sqs-daemon/sqsd"
)

func main() {

	var (
		flagSQSQueueURL       = flag.String("sqs-url", "", "The URL of the Amazon SQS queue from which messages are received.")
		flagHTTPURL           = flag.String("http-url", "http://localhost:9900/sqs", "The URL to the application that will receive the data from the Amazon SQS queue. The data is inserted into the message body of an HTTP POST message.")
		flagMIMEType          = flag.String("mime-type", "application/json", " Indicate the MIME type that the HTTP POST message uses.")
		flagHTTPTimeout       = flag.Uint("http-timeout", 59, "Timeout to wait for HTTP requests.")
		flagVisibilityTimeout = flag.Uint("visibility-timeout ", 60, "Indicate the amount of time, in seconds, an incoming message from the Amazon SQS queue is locked for processing. After the configured amount of time has passed, the message is again made visible in the queue for another daemon to read.")
		flagConnections       = flag.Uint("connections", 50, "The maximum number of concurrent connections that the daemon can make to the HTTP endpoint.")
		flagVerbose           = flag.Bool("v", false, "Log all the things")
	)

	flag.Parse()

	if *flagSQSQueueURL == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	sqsDaemon := &sqsd.Client{
		SQSQueueURL:       *flagSQSQueueURL,
		HTTPURL:           *flagHTTPURL,
		ContentType:       *flagMIMEType,
		VisibilityTimeout: int(*flagVisibilityTimeout),
		HTTPTimeout:       int(*flagHTTPTimeout),
		MaxConnections:    int(*flagConnections),
		Verbose:           *flagVerbose,
	}

	err := sqsDaemon.Start()
	if err != nil {
		log.Fatal(err)
	}

	// TODO determine if we need a stop command
	for {
		time.Sleep(time.Second)
	}
}
