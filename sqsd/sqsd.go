package sqsd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Client struct {
	SQSQueueURL       string
	HTTPURL           string
	ContentType       string
	HTTPTimeout       int
	VisibilityTimeout int
	MaxConnections    int
	Verbose           bool

	sqsClient    *sqs.SQS
	httpClient   *http.Client
	openRequests *counter
	queueName    string

	//quit         *signal
}

func (c *Client) Start() error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("error creating AWS session (check environment variables): %s", err)
	}

	c.sqsClient = sqs.New(sess)
	c.httpClient = &http.Client{Timeout: time.Duration(c.HTTPTimeout) * time.Second}
	c.openRequests = new(counter)
	//c.quit = new(signal)

	// determine queue name from url, we may need to get this from the SQS API
	c.queueName = c.SQSQueueURL[strings.LastIndex(c.SQSQueueURL, "/")+1:]

	go c.poller()
	return nil
}

func (c *Client) logf(msg string, args ...interface{}) {
	if !c.Verbose {
		return
	}
	log.Printf(msg, args...)
}

func (c *Client) poller() {

	c.logf("starting polling queue %s ...\n", c.SQSQueueURL)

	input := &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(c.SQSQueueURL),
		VisibilityTimeout:     aws.Int64(int64(c.VisibilityTimeout)),
		MaxNumberOfMessages:   aws.Int64(1),
		WaitTimeSeconds:       aws.Int64(20),
		AttributeNames:        aws.StringSlice([]string{"ApproximateFirstReceiveTimestamp", "ApproximateReceiveCount"}),
		MessageAttributeNames: aws.StringSlice([]string{"All"}),
	}

	for {

		out, err := c.sqsClient.ReceiveMessage(input)
		if err != nil {
			log.Printf("error receiving from SQS Queue: %s\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, msg := range out.Messages {

			c.logf("received queue message with ID %s\n", aws.StringValue(msg.MessageId))

			for c.openRequests.Get() >= c.MaxConnections {
				time.Sleep(10 * time.Millisecond)
			}

			c.openRequests.Add(1)
			go func() {
				defer c.openRequests.Add(-1)
				if err := c.sendHTTP(msg); err != nil {
					log.Printf("error handling message %s: %s\n", aws.StringValue(msg.MessageId), err)
				} else {
					if err := c.deleteFromQueue(msg); err != nil {
						log.Printf("error deleting message %s from queue: %s\n", aws.StringValue(msg.MessageId), err)
					}
				}
			}()

		}

	}

}

func (c *Client) deleteFromQueue(msg *sqs.Message) error {
	_, err := c.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.SQSQueueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		return err
	}

	c.logf("message %s deleted from queue", aws.StringValue(msg.MessageId))
	return nil
}

func (c *Client) sendHTTP(msg *sqs.Message) error {
	req, err := http.NewRequest(http.MethodPost, c.HTTPURL, strings.NewReader(aws.StringValue(msg.Body)))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %s", err)
	}

	// ignore time errors for now, that will just pass an empty time value
	firstReceivedTS, _ := strconv.ParseInt(aws.StringValue(msg.Attributes["ApproximateFirstReceiveTimestamp"]), 10, 64)

	req.Header.Set("User-Agent", "aws-sqsd")
	req.Header.Set("Content-Type", c.ContentType)
	req.Header.Set("X-Aws-Sqsd-Msgid", aws.StringValue(msg.MessageId))
	req.Header.Set("X-Aws-Sqsd-Queue", c.queueName)
	req.Header.Set("X-Aws-Sqsd-Receive-Count", aws.StringValue(msg.Attributes["ApproximateReceiveCount"]))
	req.Header.Set("X-Aws-Sqsd-First-Received-At", time.Unix(0, firstReceivedTS*1000000).Format(time.RFC3339))

	for attrName, attrVal := range msg.MessageAttributes {
		if aws.StringValue(attrVal.DataType) == "Binary" {
			continue
		}
		req.Header.Set("X-Aws-Sqsd-Attr-"+attrName, aws.StringValue(attrVal.StringValue))
	}

	if c.Verbose {
		reqLog, _ := httputil.DumpRequestOut(req, true)
		c.logf(string(reqLog))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.Verbose {
		respLog, _ := httputil.DumpResponse(resp, true)
		c.logf(string(respLog))
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %s", err)
	}

	return fmt.Errorf("received non 200 HTTP response code %s: %s", resp.Status, string(b))
}
