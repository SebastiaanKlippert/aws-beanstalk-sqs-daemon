# aws-beanstalk-sqs-daemon
AWS Elastic Beanstalk SQS Daemon simulator

This project will simultate the SQS deamon which sends the queue messages
to a HTTP endpoint as described in 
https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/using-features-managing-env-tiers.html

WIP, no support, no guarantees.

![aeb-messageflow-worker.png](
https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/images/aeb-messageflow-worker.png)

## Commandline flags
Only sqs-url is required.
```
  -connections uint
        The maximum number of concurrent connections that the daemon can make to the HTTP endpoint. (default 50)
  -http-timeout uint
        Timeout in seconds to wait for HTTP requests. (default 30)
  -http-url string
        The URL to the application that will receive the data from the Amazon SQS queue. The data is inserted into the message body of an HTTP POST message. (default "http://localhost:9900/sqs")
  -mime-type string
         Indicate the MIME type that the HTTP POST message uses. (default "application/json")
  -sqs-url string
        The URL of the Amazon SQS queue from which messages are received.
  -v    Log all the things
  -visibility-timeout  uint
        Indicate the amount of time, in seconds, an incoming message from the Amazon SQS queue is locked for processing. After the configured amount of time has passed, the message is again made visible in the queue for another daemon to read. (default 60)
```