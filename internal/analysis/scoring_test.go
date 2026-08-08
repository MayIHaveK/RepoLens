package analysis

import (
	"math"
	"testing"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

func TestCompositeSharesSumToHundred(t *testing.T) {
	agg := newAggregate()
	agg.contributors["a"] = &contributorAggregate{
		Contributor: model.Contributor{ID: "a", Name: "A", Additions: 100, HistoricalAdded: 100, OwnedLines: 80, Categories: map[model.Category]model.Change{}},
		Workload:    100, Directories: map[string]model.Change{},
	}
	agg.contributors["b"] = &contributorAggregate{
		Contributor: model.Contributor{ID: "b", Name: "B", Additions: 50, HistoricalAdded: 50, OwnedLines: 20, Categories: map[model.Category]model.Change{}},
		Workload:    50, Directories: map[string]model.Change{},
	}
	result := finalizeContributors(agg, config.Default())
	total := result[0].CompositeShare + result[1].CompositeShare
	if math.Abs(total-100) > 0.02 {
		t.Fatalf("composite shares should sum to 100, got %.2f", total)
	}
}

func TestSplitCreditPreservesTotal(t *testing.T) {
	var total int64
	for i := 0; i < 3; i++ {
		total += splitCredit(10, 3, i)
	}
	if total != 10 {
		t.Fatalf("credit split lost lines: got %d", total)
	}
}
