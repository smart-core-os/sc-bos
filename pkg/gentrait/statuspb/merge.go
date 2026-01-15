package statuspb

import (
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

type ProblemMerger struct {
	BiggestProblem    *statuspb.StatusLog_Problem
	MostRecentNominal *statuspb.StatusLog_Problem

	problems []*statuspb.StatusLog_Problem
}

func (pm *ProblemMerger) Empty() bool {
	return len(pm.problems) == 0 && pm.BiggestProblem == nil && pm.MostRecentNominal == nil
}

func (pm *ProblemMerger) Build() *statuspb.StatusLog {
	out := &statuspb.StatusLog{}
	if pm.BiggestProblem == nil {
		// no problems
		out.Level = statuspb.StatusLog_NOMINAL
		if pm.MostRecentNominal != nil {
			out.Level = pm.MostRecentNominal.Level
			out.Description = pm.MostRecentNominal.Description
			out.RecordTime = pm.MostRecentNominal.RecordTime
		}
	} else {
		out.Level = pm.BiggestProblem.Level
		out.Description = pm.BiggestProblem.Description
		out.RecordTime = pm.BiggestProblem.RecordTime

		if pm.BiggestProblem.Level == statuspb.StatusLog_OFFLINE && pm.MostRecentNominal != nil {
			out.Level = statuspb.StatusLog_REDUCED_FUNCTION
		}
	}
	out.Problems = pm.problems // this reuses a reference, but for our use I'm sure it's fine
	return out
}

func (pm *ProblemMerger) AddStatusLog(sl *statuspb.StatusLog) {
	pm.AddProblems(sl.Problems)
	if sl.Level == statuspb.StatusLog_NOMINAL {
		switch {
		case pm.MostRecentNominal == nil:
			pm.MostRecentNominal = &statuspb.StatusLog_Problem{
				Level:       statuspb.StatusLog_NOMINAL,
				Description: sl.Description,
				RecordTime:  sl.RecordTime,
			}
		case sl.RecordTime == nil: // do nothing
		case pm.MostRecentNominal.RecordTime == nil:
			pm.MostRecentNominal.RecordTime = sl.RecordTime
			pm.MostRecentNominal.Description = sl.Description
		case sl.RecordTime.AsTime().After(pm.MostRecentNominal.RecordTime.AsTime()):
			pm.MostRecentNominal.RecordTime = sl.RecordTime
			pm.MostRecentNominal.Description = sl.Description
		}
	}
}

func (pm *ProblemMerger) AddProblemMessages(problems []proto.Message) {
	for _, problem := range problems {
		pm.AddProblem(problem.(*statuspb.StatusLog_Problem))
	}
}

func (pm *ProblemMerger) AddProblems(problems []*statuspb.StatusLog_Problem) {
	for _, problem := range problems {
		pm.AddProblem(problem)
	}
}

func (pm *ProblemMerger) AddProblem(problem *statuspb.StatusLog_Problem) {
	if problem.Level == statuspb.StatusLog_NOMINAL {
		switch {
		case pm.MostRecentNominal == nil:
			pm.MostRecentNominal = problem
		case pm.MostRecentNominal.RecordTime == nil:
			pm.MostRecentNominal = problem
		case problem.RecordTime == nil:
		case problem.RecordTime.AsTime().After(pm.MostRecentNominal.RecordTime.AsTime()):
			pm.MostRecentNominal = problem
		}
		return
	}
	pm.problems = append(pm.problems, problem)
	if pm.BiggestProblem == nil || problem.Level > pm.BiggestProblem.Level {
		pm.BiggestProblem = problem
	}
	if pm.BiggestProblem.Level == problem.Level && problem.RecordTime != nil {
		// make sure we're reporting the earliest problem at this level
		switch {
		case pm.BiggestProblem.RecordTime == nil:
			pm.BiggestProblem = problem
		case problem.RecordTime.AsTime().Before(pm.BiggestProblem.RecordTime.AsTime()):
			pm.BiggestProblem = problem
		}
	}
}
