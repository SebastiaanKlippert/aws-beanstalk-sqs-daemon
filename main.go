package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/SebastiaanKlippert/aws-beanstalk-sqs-daemon/createqueue"
	"github.com/SebastiaanKlippert/aws-beanstalk-sqs-daemon/sqsd"
)

func main() {

	var (
		flagSQSQueueURL        = flag.String("sqs-url", "", "The URL of the Amazon SQS queue from which messages are received. Use this or create-queue.")
		flagCreateQueueName    = flag.String("sqs-create-queue", "", "Creates a queue with this name, subscribes it to the SNS topics listed in subscribe-to-sns-arns and then uses this queue to receive messages. Use this or sqs-url.")
		flagSubscribeToSNSARNs = flag.String("subscribe-to-sns-arns", "", "Comma separated list of SNS topic ARNs to subscribe the created queue to (for existing queues no new subscriptions will be added).")
		flagHTTPURL            = flag.String("http-url", "http://localhost:9900/sqs", "The URL to the application that will receive the data from the Amazon SQS queue. The data is inserted into the message body of an HTTP POST message.")
		flagMIMEType           = flag.String("mime-type", "application/json", " Indicate the MIME type that the HTTP POST message uses.")
		flagHTTPTimeout        = flag.Uint("http-timeout", 30, "Timeout in seconds to wait for HTTP requests.")
		flagVisibilityTimeout  = flag.Uint("visibility-timeout ", 60, "Indicate the amount of time, in seconds, an incoming message from the Amazon SQS queue is locked for processing. After the configured amount of time has passed, the message is again made visible in the queue for another daemon to read.")
		flagConnections        = flag.Uint("connections", 50, "The maximum number of concurrent connections that the daemon can make to the HTTP endpoint.")

		flagVerbose = flag.Bool("v", false, "Log all the things.")
	)

	flag.Parse()

	if *flagSQSQueueURL == "" {
		*flagSQSQueueURL = os.Getenv("SQS_URL")
	}
	if *flagCreateQueueName == "" {
		*flagCreateQueueName = os.Getenv("SQS_CREATE_QUEUE")
	}

	if *flagSQSQueueURL == "" && *flagCreateQueueName == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *flagCreateQueueName != "" {
		// create the queue and subscribe it first

		queueARNs := strings.Split(*flagSubscribeToSNSARNs, ",")
		if len(queueARNs) == 1 && len(queueARNs[0]) == 0 {
			queueARNs = make([]string, 0)
		}

		createOptions := &createqueue.CreateOptions{
			QueueName:         *flagCreateQueueName,
			SNSTopicARNs:      queueARNs,
			VisibilityTimeout: int(*flagVisibilityTimeout),
			Verbose:           *flagVerbose,
		}
		createOutput, err := createqueue.CreateAndSubscribe(createOptions)
		if err != nil {
			log.Fatal(err)
		}
		*flagSQSQueueURL = createOutput.SQSQueueURL
	}

	// start the SQS daemon client
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
