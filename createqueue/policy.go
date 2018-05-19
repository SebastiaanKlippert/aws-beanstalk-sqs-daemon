package createqueue

import (
	"encoding/json"
	"fmt"
)

type policyDocument struct {
	Version   string            `json:"Version"`
	ID        string            `json:"Id"`
	Statement []policyStatement `json:"Statement"`
}

type policyStatement struct {
	Sid       string `json:"Sid"`
	Effect    string `json:"Effect"`
	Principal struct {
		AWS string `json:"AWS"`
	} `json:"Principal"`
	Action    string `json:"Action"`
	Resource  string `json:"Resource"`
	Condition struct {
		ArnEquals struct {
			AwsSourceArn []string `json:"aws:SourceArn"`
		} `json:"ArnEquals"`
	} `json:"Condition"`
}

// NewSQSSendPolicyForSNSSourceARNs returns the SQS Send Policy
func NewSQSSendPolicyForSNSSourceARNs(sqsARN string, snsARNs []string) (string, error) {

	stmt := policyStatement{
		Effect:   "Allow",
		Action:   "SQS:SendMessage",
		Resource: sqsARN,
	}

	stmt.Principal.AWS = "*"
	stmt.Condition.ArnEquals.AwsSourceArn = snsARNs

	p := &policyDocument{
		Version:   "2012-10-17",
		ID:        fmt.Sprintf("%s/SQSDefaultPolicy", sqsARN),
		Statement: []policyStatement{stmt},
	}

	buf, err := json.Marshal(p)
	return string(buf), err
}
