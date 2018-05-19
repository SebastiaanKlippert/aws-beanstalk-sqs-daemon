package createqueue

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// CreateOptions are the options to create and subscribe the SQS queue
type CreateOptions struct {
	QueueName         string
	SNSTopicARNs      []string
	VisibilityTimeout int
	Verbose           bool
}

func (opts *CreateOptions) logf(msg string, args ...interface{}) {
	if !opts.Verbose {
		return
	}
	log.Printf(msg, args...)
}

// CreateAndSubscribe creates the SQS queue and subscribes it to the SNS topics
func CreateAndSubscribe(opts *CreateOptions) (string, error) {

	sqsQueueURL := ""

	opts.logf("Creating SQS queue %s...", opts.QueueName)

	sess, err := session.NewSession()
	if err != nil {
		return "", fmt.Errorf("error creating AWS session (check environment variables): %s", err)
	}

	// Create SQS service
	sqsService := sqs.New(sess)

	opts.logf("Listing existing queues...")

	// List all SQS queues beginning with the same name
	// and select the correct queue or create a new queue
	lqi := &sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(opts.QueueName),
	}
	listResult, err := sqsService.ListQueues(lqi)
	if err != nil {
		return "", fmt.Errorf("error listing SQS queues: %s", err)
	}

	// Find the exact queue name
	for _, q := range listResult.QueueUrls {
		if strings.HasSuffix(aws.StringValue(q), "/"+opts.QueueName) {
			sqsQueueURL = aws.StringValue(q)
			break
		}
	}
	if len(sqsQueueURL) > 0 {
		// The queue already exists, we are done
		opts.logf("Using existing SQS queue with URL %s", sqsQueueURL)
		return sqsQueueURL, nil
	}

	// There is no SQS queue with this name yet, create it
	cqi := &sqs.CreateQueueInput{
		QueueName: aws.String(opts.QueueName),
	}

	// A recently deleted queue cannot be recreated immediately, for safety we will build a retry mechanism here
	for nTries := 0; nTries < 12; nTries++ {

		createResponse, err := sqsService.CreateQueue(cqi)
		if err != nil {
			if strings.HasPrefix(err.Error(), "AWS.SimpleQueueService.QueueDeletedRecently") {
				opts.logf("Waiting 10 seconds for recently deleted queue to become available again...")
				time.Sleep(10 * time.Second)
				continue
			} else {
				return "", fmt.Errorf("error creating SQS queue: %s", err)
			}
		}
		sqsQueueURL = aws.StringValue(createResponse.QueueUrl)
	}

	opts.logf("Created SQS status queue with URL %s", sqsQueueURL)

	// We now need to get the ARN of the created status queue
	gqai := &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(sqsQueueURL),
		AttributeNames: aws.StringSlice([]string{"QueueArn"}),
	}
	queueAttributes, err := sqsService.GetQueueAttributes(gqai)
	if err != nil {
		return sqsQueueURL, fmt.Errorf("error getting SQS queue attributes: %s", err)
	}
	sqsQueueARN := aws.StringValue(queueAttributes.Attributes["QueueArn"])

	// We now create the SNS service for each topic because the regions can differ and subscribe the new queue to all the SNS topics
	for _, topicARN := range opts.SNSTopicARNs {

		if err := subscribeQueueToSNSTopic(sess, sqsQueueARN, topicARN); err != nil {
			return sqsQueueURL, err
		}
		opts.logf("Subscribed SQS ARN %s to SNS topic ARN %s", sqsQueueARN, topicARN)
	}

	// We now need to set the required queue attributes and policy
	// we replace the ARNs in the policy document
	policy, err := NewSQSSendPolicyForSNSSourceARNs(sqsQueueARN, opts.SNSTopicARNs)
	if err != nil {
		return sqsQueueURL, fmt.Errorf("error creating SQS queue policy JSON document: %s", err)
	}

	// Set queue attributes
	qAttrs := map[string]string{
		"Policy":                        policy,
		"VisibilityTimeout":             strconv.Itoa(opts.VisibilityTimeout),
		"ReceiveMessageWaitTimeSeconds": "20",
	}

	sqai := &sqs.SetQueueAttributesInput{
		QueueUrl:   aws.String(sqsQueueURL),
		Attributes: aws.StringMap(qAttrs),
	}
	_, err = sqsService.SetQueueAttributes(sqai)
	if err != nil {
		return sqsQueueURL, fmt.Errorf("error setting SQS queue attributes: %s", err)
	}

	opts.logf("Queue attributes set successfully, queue creation is now complete")

	return sqsQueueURL, nil
}

func subscribeQueueToSNSTopic(sess *session.Session, sqsQueueARN, topicARN string) error {

	ARN, err := arn.Parse(topicARN)
	if err != nil {
		return fmt.Errorf("error parsing SNS topic ARN %s: %s", topicARN, err)
	}

	snsConfig := aws.NewConfig().WithRegion(ARN.Region)
	snsService := sns.New(sess, snsConfig)

	si := &sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		Endpoint: aws.String(sqsQueueARN),
		TopicArn: aws.String(topicARN),
	}
	subscribeResult, err := snsService.Subscribe(si)
	if err != nil {
		return fmt.Errorf("error subscribing to SNS topic ARN %s: %s", topicARN, err)
	}

	// Set RAW message delivery on to receive SQS style messages
	// See http://docs.aws.amazon.com/sns/latest/dg/large-payload-raw-message.html
	ssai := &sns.SetSubscriptionAttributesInput{
		SubscriptionArn: subscribeResult.SubscriptionArn,
		AttributeName:   aws.String("RawMessageDelivery"),
		AttributeValue:  aws.String("true"),
	}
	_, err = snsService.SetSubscriptionAttributes(ssai)
	if err != nil {
		return fmt.Errorf("error in SNS SetSubscriptionAttributes for topic ARN %s: %s", topicARN, err)
	}

	return nil
}
