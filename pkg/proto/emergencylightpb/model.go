package emergencylightpb

import (
	"context"
	"math/rand"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type Model struct {
	testResultSet *resource.Value // of *emergencylightpb.TestResultSet
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&TestResultSet{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		testResultSet: resource.NewValue(opts...),
	}
}

func (m *Model) SetLastDurationTest(result EmergencyTestResult_Result) {
	_, _ = m.testResultSet.Set(&TestResultSet{
		DurationTest: &EmergencyTestResult{
			EndTime: timestamppb.Now(),
			Result:  result,
		}}, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
		Paths: []string{"duration_test"},
	}))
}

func (m *Model) SetLastFunctionalTest(result EmergencyTestResult_Result) {
	_, _ = m.testResultSet.Set(&TestResultSet{
		FunctionTest: &EmergencyTestResult{
			EndTime: timestamppb.Now(),
			Result:  result,
		}}, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
		Paths: []string{"function_test"},
	}))
}

func (m *Model) GetTestResultSet() *TestResultSet {
	return m.testResultSet.Get().(*TestResultSet)
}

func (m *Model) RunDurationTest() {
	result := &EmergencyTestResult{
		EndTime: timestamppb.Now(),
		Result:  getRandomEmergencyLightResult(),
	}
	m.SetLastDurationTest(result.Result)
}

func (m *Model) RunFunctionTest() {
	result := &EmergencyTestResult{
		EndTime: timestamppb.Now(),
		Result:  getRandomEmergencyLightResult(),
	}
	m.SetLastFunctionalTest(result.Result)
}

func getRandomEmergencyLightResult() EmergencyTestResult_Result {
	n := rand.Intn(11)
	return EmergencyTestResult_Result(n)
}

func (m *Model) PullTestResults(ctx context.Context, opts ...resource.ReadOption) <-chan PullTestResultSetChange {
	return resources.PullValue[*TestResultSet](ctx, m.testResultSet.Pull(ctx, opts...))
}

type PullTestResultSetChange = resources.ValueChange[*TestResultSet]
