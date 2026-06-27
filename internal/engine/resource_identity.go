package engine

import "strconv"

type duplicateResourceID struct {
	id         string
	typ        string
	index      int
	firstIndex int
}

func duplicateResourceIDPlanItems(resources []Resource) []PlanItem {
	duplicates := findDuplicateResourceIDs(resources)
	items := make([]PlanItem, 0, len(duplicates))
	for _, duplicate := range duplicates {
		message := duplicateResourceIDMessage(duplicate.id)
		items = append(items, PlanItem{
			ResourceID: duplicate.id,
			Type:       duplicate.typ,
			State:      StateFailed,
			Action:     ActionFail,
			Message:    message,
			Error:      message,
			Details:    duplicateResourceIDDetails(duplicate),
		})
	}
	return items
}

func duplicateResourceIDApplyItems(resources []Resource) []ApplyItem {
	duplicates := findDuplicateResourceIDs(resources)
	items := make([]ApplyItem, 0, len(duplicates))
	for _, duplicate := range duplicates {
		message := duplicateResourceIDMessage(duplicate.id)
		items = append(items, ApplyItem{
			ResourceID: duplicate.id,
			Type:       duplicate.typ,
			Action:     "fail",
			Message:    message,
			Error:      message,
			Details:    duplicateResourceIDDetails(duplicate),
		})
	}
	return items
}

func findDuplicateResourceIDs(resources []Resource) []duplicateResourceID {
	firstByID := make(map[string]int, len(resources))
	firstReported := make(map[string]bool)
	duplicates := make([]duplicateResourceID, 0)

	for i, resource := range resources {
		id := resource.ID()
		if firstIndex, ok := firstByID[id]; ok {
			if !firstReported[id] {
				duplicates = append(duplicates, duplicateResourceID{
					id:         id,
					typ:        resources[firstIndex].Type(),
					index:      firstIndex,
					firstIndex: firstIndex,
				})
				firstReported[id] = true
			}
			duplicates = append(duplicates, duplicateResourceID{
				id:         id,
				typ:        resource.Type(),
				index:      i,
				firstIndex: firstIndex,
			})
			continue
		}

		firstByID[id] = i
	}

	return duplicates
}

func duplicateResourceIDMessage(id string) string {
	return "duplicate resource ID " + strconv.Quote(id) + "; resource IDs must be unique"
}

func duplicateResourceIDDetails(duplicate duplicateResourceID) map[string]string {
	return map[string]string{
		"resource_index": strconv.Itoa(duplicate.index),
		"first_index":    strconv.Itoa(duplicate.firstIndex),
	}
}
