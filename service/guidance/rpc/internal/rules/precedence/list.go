package precedence

import (
	"context"
	"sort"

	"hospital/common/authn"
)

type ListInput struct {
	ItemID          string
	Direction       Direction
	IncludeInferred bool
}

func (m *Manager) List(ctx context.Context, operator authn.Principal, input ListInput) ([]Relation, error) {
	if err := requireRead(operator); err != nil {
		return nil, err
	}
	itemID, err := normalizeUUID(input.ItemID)
	if err != nil {
		return nil, err
	}
	direction, err := normalizeDirection(input.Direction)
	if err != nil {
		return nil, err
	}
	item, err := m.projects.ResolveProject(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err := requireDepartmentScope(operator, item.DepartmentID); err != nil {
		return nil, err
	}
	rules, err := m.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := relationsForItem(rules, itemID, direction, input.IncludeInferred)
	sort.Slice(result, func(i, j int) bool {
		if result[i].PathLength != result[j].PathLength {
			return result[i].PathLength < result[j].PathLength
		}
		if result[i].PredecessorItemName != result[j].PredecessorItemName {
			return result[i].PredecessorItemName < result[j].PredecessorItemName
		}
		return result[i].SuccessorItemName < result[j].SuccessorItemName
	})
	return result, nil
}

func reachable(rules []Rule, start, target string) bool {
	if start == target {
		return true
	}
	adjacency := make(map[string][]string)
	for _, rule := range rules {
		adjacency[rule.PredecessorItemID] = append(adjacency[rule.PredecessorItemID], rule.SuccessorItemID)
	}
	seen := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if next == target {
				return true
			}
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

func relationsForItem(rules []Rule, itemID string, direction Direction, includeInferred bool) []Relation {
	direct := make(map[string]Relation)
	projects := make(map[string]ProjectReference)
	for _, rule := range rules {
		projects[rule.PredecessorItemID] = ProjectReference{ItemID: rule.PredecessorItemID, DepartmentID: rule.PredecessorDepartmentID, Name: rule.PredecessorItemName}
		projects[rule.SuccessorItemID] = ProjectReference{ItemID: rule.SuccessorItemID, DepartmentID: rule.SuccessorDepartmentID, Name: rule.SuccessorItemName}
		if (direction == DirectionAll || direction == DirectionPredecessors) && rule.SuccessorItemID == itemID {
			direct[rule.PredecessorItemID+">"+itemID] = Relation{Rule: rule, Direct: true, PathLength: 1}
		}
		if (direction == DirectionAll || direction == DirectionSuccessors) && rule.PredecessorItemID == itemID {
			direct[itemID+">"+rule.SuccessorItemID] = Relation{Rule: rule, Direct: true, PathLength: 1}
		}
	}
	if includeInferred {
		if direction == DirectionAll || direction == DirectionPredecessors {
			for node, distance := range shortestDistances(rules, itemID, true) {
				if distance < 2 {
					continue
				}
				from, to := projects[node], projects[itemID]
				direct[node+">"+itemID] = inferredRelation(from, to, distance)
			}
		}
		if direction == DirectionAll || direction == DirectionSuccessors {
			for node, distance := range shortestDistances(rules, itemID, false) {
				if distance < 2 {
					continue
				}
				from, to := projects[itemID], projects[node]
				direct[itemID+">"+node] = inferredRelation(from, to, distance)
			}
		}
	}
	result := make([]Relation, 0, len(direct))
	for _, relation := range direct {
		result = append(result, relation)
	}
	return result
}

func shortestDistances(rules []Rule, start string, reverse bool) map[string]int {
	adjacency := make(map[string][]string)
	for _, rule := range rules {
		from, to := rule.PredecessorItemID, rule.SuccessorItemID
		if reverse {
			from, to = to, from
		}
		adjacency[from] = append(adjacency[from], to)
	}
	distances := map[string]int{start: 0}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if _, exists := distances[next]; exists {
				continue
			}
			distances[next] = distances[current] + 1
			queue = append(queue, next)
		}
	}
	delete(distances, start)
	return distances
}

func inferredRelation(from, to ProjectReference, distance int) Relation {
	return Relation{Rule: Rule{
		PredecessorItemID: from.ItemID, PredecessorDepartmentID: from.DepartmentID, PredecessorItemName: from.Name,
		SuccessorItemID: to.ItemID, SuccessorDepartmentID: to.DepartmentID, SuccessorItemName: to.Name,
	}, Direct: false, PathLength: distance}
}
