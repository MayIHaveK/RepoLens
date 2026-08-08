package analysis

import (
	"sort"

	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/model"
)

func finalizeContributors(agg *aggregate, cfg config.Config) []model.Contributor {
	var totalWorkload, totalOwnership, totalRetention, totalCollaboration float64
	for _, contributor := range agg.contributors {
		totalWorkload += contributor.Workload
		totalOwnership += float64(contributor.Contributor.OwnedLines)
		retentionRate := 0.0
		if contributor.Contributor.HistoricalAdded > 0 {
			retentionRate = minFloat(float64(contributor.Contributor.OwnedLines)/float64(contributor.Contributor.HistoricalAdded), 1)
		}
		contributor.Contributor.RetentionRate = round(retentionRate*100, 1)
		totalRetention += float64(contributor.Contributor.OwnedLines) * retentionRate
		totalCollaboration += float64(contributor.Contributor.Reviews*3 + contributor.Contributor.Issues)
	}

	availableWeight := cfg.Weights.Workload
	if totalOwnership > 0 && cfg.EnableOwnership {
		availableWeight += cfg.Weights.Ownership
	}
	if totalRetention > 0 && cfg.EnableRetention {
		availableWeight += cfg.Weights.Retention
	}
	if totalCollaboration > 0 && cfg.EnableCollaboration {
		availableWeight += cfg.Weights.Collaboration
	}
	if availableWeight <= 0 {
		availableWeight = 1
	}

	result := make([]model.Contributor, 0, len(agg.contributors))
	for _, aggregate := range agg.contributors {
		contributor := &aggregate.Contributor
		if totalWorkload > 0 {
			contributor.WorkloadShare = aggregate.Workload / totalWorkload * 100
		}
		if totalOwnership > 0 {
			contributor.OwnershipShare = float64(contributor.OwnedLines) / totalOwnership * 100
		}
		if totalRetention > 0 {
			retentionRate := contributor.RetentionRate / 100
			contributor.RetentionShare = float64(contributor.OwnedLines) * retentionRate / totalRetention * 100
		}
		if totalCollaboration > 0 {
			points := float64(contributor.Reviews*3 + contributor.Issues)
			contributor.CollaborationShare = points / totalCollaboration * 100
		}

		composite := contributor.WorkloadShare * cfg.Weights.Workload
		if totalOwnership > 0 && cfg.EnableOwnership {
			composite += contributor.OwnershipShare * cfg.Weights.Ownership
		}
		if totalRetention > 0 && cfg.EnableRetention {
			composite += contributor.RetentionShare * cfg.Weights.Retention
		}
		if totalCollaboration > 0 && cfg.EnableCollaboration {
			composite += contributor.CollaborationShare * cfg.Weights.Collaboration
		}
		contributor.CompositeShare = composite / availableWeight
		contributor.WorkloadShare = round(contributor.WorkloadShare, 2)
		contributor.OwnershipShare = round(contributor.OwnershipShare, 2)
		contributor.RetentionShare = round(contributor.RetentionShare, 2)
		contributor.CollaborationShare = round(contributor.CollaborationShare, 2)
		contributor.CompositeShare = round(contributor.CompositeShare, 2)
		contributor.TopDirectories = topDirectories(aggregate.Directories, cfg.Privacy.ShowDirectories)
		sort.SliceStable(contributor.RecentCommits, func(i, j int) bool {
			return contributor.RecentCommits[i].Timestamp > contributor.RecentCommits[j].Timestamp
		})
		if len(contributor.RecentCommits) > 8 {
			contributor.RecentCommits = contributor.RecentCommits[:8]
		}
		result = append(result, *contributor)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CompositeShare == result[j].CompositeShare {
			return result[i].Name < result[j].Name
		}
		return result[i].CompositeShare > result[j].CompositeShare
	})
	return result
}

func topDirectories(values map[string]model.Change, enabled bool) []model.DirectoryShare {
	if !enabled {
		return nil
	}
	result := make([]model.DirectoryShare, 0, len(values))
	for name, change := range values {
		result = append(result, model.DirectoryShare{Name: name, Additions: change.Additions, Deletions: change.Deletions})
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i].Additions + result[i].Deletions
		right := result[j].Additions + result[j].Deletions
		return left > right
	})
	if len(result) > 6 {
		result = result[:6]
	}
	return result
}

func finalizeCategories(agg *aggregate) []model.CategorySummary {
	for _, contributor := range agg.contributors {
		for category, change := range contributor.Contributor.Categories {
			summary := agg.categories[category]
			if summary == nil {
				summary = &model.CategorySummary{Category: category}
				agg.categories[category] = summary
			}
			summary.Additions += change.Additions
			summary.Deletions += change.Deletions
		}
	}
	order := []model.Category{
		model.CategorySource, model.CategoryTest, model.CategoryDocs,
		model.CategoryConfig, model.CategoryAsset, model.CategoryOther,
	}
	result := make([]model.CategorySummary, 0, len(order))
	for _, category := range order {
		if summary := agg.categories[category]; summary != nil {
			result = append(result, *summary)
		}
	}
	return result
}
