package util

import (
	"strings"
)

const JobKeySep = "|"

func JobKey(machineID, jobName string) string {
	return machineID + JobKeySep + jobName
}

func SplitJobKey(key string) (machineID, jobName string) {
	if i := strings.Index(key, JobKeySep); i >= 0 {
		return key[:i], key[i+1:]
	}
	return key, ""
}

func SanitizeJobName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "job-" + RandomString(8)
	}
	return name
}