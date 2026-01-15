package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
)

func Test_processIngressPermitted(t *testing.T) {
	// Case 1: Correct string input, with IngressPermittedType
	typeStr := "badge"
	cfg := accessConfig{IngressPermittedType: &typeStr}
	data := &accesspb.AccessAttempt{}
	err := processIngressPermitted("12345", data, cfg)
	assert.NoError(t, err)
	assert.Equal(t, accesspb.AccessAttempt_GRANTED, data.Grant)
	assert.NotNil(t, data.AccessAttemptTime)
	assert.NotNil(t, data.Actor)
	assert.Equal(t, map[string]string{"badge": "12345"}, data.Actor.Ids)
	assert.NotNil(t, data.Actor.LastGrantTime)

	// Case 2: Correct string input, without IngressPermittedType
	cfg2 := accessConfig{}
	data2 := &accesspb.AccessAttempt{}
	err = processIngressPermitted("67890", data2, cfg2)
	assert.NoError(t, err)
	assert.Equal(t, accesspb.AccessAttempt_GRANTED, data2.Grant)
	assert.NotNil(t, data2.AccessAttemptTime)
	assert.NotNil(t, data2.Actor)
	assert.Nil(t, data2.Actor.Ids)
	assert.NotNil(t, data2.Actor.LastGrantTime)

	// Case 3: Incorrect type (int)
	cfg3 := accessConfig{}
	data3 := &accesspb.AccessAttempt{}
	err = processIngressPermitted(12345, data3, cfg3)
	assert.Error(t, err)
}

func Test_processIngressDenied(t *testing.T) {
	// Case 1: Correct string input, with IngressDeniedType
	typeStr := "badge"
	cfg := accessConfig{IngressDeniedType: &typeStr}
	data := &accesspb.AccessAttempt{}
	err := processIngressDenied("54321", data, cfg)
	assert.NoError(t, err)
	assert.Equal(t, accesspb.AccessAttempt_DENIED, data.Grant)
	assert.NotNil(t, data.AccessAttemptTime)
	assert.NotNil(t, data.Actor)
	assert.Equal(t, map[string]string{"badge": "54321"}, data.Actor.Ids)

	// Case 2: Correct string input, without IngressDeniedType
	cfg2 := accessConfig{}
	data2 := &accesspb.AccessAttempt{}
	err = processIngressDenied("09876", data2, cfg2)
	assert.NoError(t, err)
	assert.Equal(t, accesspb.AccessAttempt_DENIED, data2.Grant)
	assert.NotNil(t, data2.AccessAttemptTime)
	assert.Nil(t, data2.Actor)

	// Case 3: Incorrect type (int)
	cfg3 := accessConfig{}
	data3 := &accesspb.AccessAttempt{}
	err = processIngressDenied(54321, data3, cfg3)
	assert.Error(t, err)
}
