package utils

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// GetInstanceName extracts the "Name" tag from an EC2 instance.
func GetInstanceName(instance *ec2.Instance) string {
	for _, tag := range instance.Tags {
		if aws.StringValue(tag.Key) == "Name" {
			return aws.StringValue(tag.Value)
		}
	}
	return "N/A"
}

// GetSecurityGroupNames extracts security group names from an EC2 instance.
func GetSecurityGroupNames(sgs []*ec2.SecurityGroupIdentifier) string {
	var names []string
	for _, sg := range sgs {
		names = append(names, aws.StringValue(sg.GroupName))
	}
	if len(names) == 0 {
		return "N/A"
	}
	return strings.Join(names, ", ")
}

func ArrayToCSV(array []*string) string {
	ret := ""
	for i, v := range array {
		ret += aws.StringValue(v)
		if i < len(array)-1 {
			ret += ", "

		}
	}
	return ret
}
