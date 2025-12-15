package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOriginalCallerFuncName(t *testing.T) {
	caller := GetOriginalCallerFuncName(2)
	assert.Equal(t,
		"github.com/MikL9/observability/utils.TestGetOriginalCallerFuncName",
		caller,
	)
}

func TestGetOpNameBySnakeCase(t *testing.T) {
	testCases := map[string]string{
		"da-data-proxy/internal/service.(*dadataService).ParseAddress": "parse_address",
		"observability.TestGetOpName":                                  "test_get_op_name",
		"wb_sender.SMSStatusBRMCode":                                   "sms_status_brm_code",
		"GetS3Files":                                                   "get_s3_files",
	}
	for input, expected := range testCases {
		output := GetOpNameBySnakeCase(input)
		assert.Equal(t, expected, output)
	}
}

func TestGetOpName(t *testing.T) {
	testCases := map[string]string{
		"da-data-proxy/internal/service.(*dadataService).ParseAddress": "parse address",
		"observability.TestGetOpName":                                  "test get op name",
		"wb_sender.SMSStatusBRMCode":                                   "sms status brm code",
		"GetS3Files":                                                   "get s3 files",
	}
	for input, expected := range testCases {
		output := GetOpName(input)
		assert.Equal(t, expected, output)
	}
}
