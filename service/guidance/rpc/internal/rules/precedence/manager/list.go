package manager

import (
	"context"
	"sort"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type ListInput struct {
	ItemID          string
	Direction       precedence.Direction
	IncludeInferred bool
}

func (m *Manager) List(ctx context.Context, operator authn.Principal, input ListInput) ([]precedence.Relation, error) {
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

func reachable(rules []precedence.Rule, start, target string) bool {
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

func relationsForItem(rules []precedence.Rule, itemID string, direction precedence.Direction, includeInferred bool) []precedence.Relation {
	direct := make(map[string]precedence.Relation)
	projects := make(map[string]precedence.ProjectReference)
	for _, rule := range rules {
		projects[rule.PredecessorItemID] = precedence.ProjectReference{ItemID: rule.PredecessorItemID, DepartmentID: rule.PredecessorDepartmentID, Name: rule.PredecessorItemName}
		projects[rule.SuccessorItemID] = precedence.ProjectReference{ItemID: rule.SuccessorItemID, DepartmentID: rule.SuccessorDepartmentID, Name: rule.SuccessorItemName}
		if (direction == precedence.DirectionAll || direction == precedence.DirectionPredecessors) && rule.SuccessorItemID == itemID {
			direct[rule.PredecessorItemID+">"+itemID] = precedence.Relation{Rule: rule, Direct: true, PathLength: 1}
		}
		if (direction == precedence.DirectionAll || direction == precedence.DirectionSuccessors) && rule.PredecessorItemID == itemID {
			direct[itemID+">"+rule.SuccessorItemID] = precedence.Relation{Rule: rule, Direct: true, PathLength: 1}
		}
	}
	if includeInferred {
		if direction == precedence.DirectionAll || direction == precedence.DirectionPredecessors {
			for node, distance := range shortestDistances(rules, itemID, true) {
				if distance < 2 {
					continue
				}
				from, to := projects[node], projects[itemID]
				direct[node+">"+itemID] = inferredRelation(from, to, distance)
			}
		}
		if direction == precedence.DirectionAll || direction == precedence.DirectionSuccessors {
			for node, distance := range shortestDistances(rules, itemID, false) {
				if distance < 2 {
					continue
				}
				from, to := projects[itemID], projects[node]
				direct[itemID+">"+node] = inferredRelation(from, to, distance)
			}
		}
	}
	result := make([]precedence.Relation, 0, len(direct))
	for _, relation := range direct {
		result = append(result, relation)
	}
	return result
}

func shortestDistances(rules []precedence.Rule, start string, reverse bool) map[string]int {
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

func inferredRelation(from, to precedence.ProjectReference, distance int) precedence.Relation {
	return precedence.Relation{Rule: precedence.Rule{
		PredecessorItemID: from.ItemID, PredecessorDepartmentID: from.DepartmentID, PredecessorItemName: from.Name,
		SuccessorItemID: to.ItemID, SuccessorDepartmentID: to.DepartmentID, SuccessorItemName: to.Name,
	}, Direct: false, PathLength: distance}
}
