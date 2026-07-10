package fronts

import (
	"math"
	"sort"
	"time"

	mongomodel "github.com/sitcon-tw/camp2026-game/internal/mongodb/model"
)

const (
	frontRailNeighborLimit = 2
	frontRailMaxDistance   = 16
)

type frontRailNeighbor struct {
	id       string
	distance int
}

func reconcileFrontRailNetwork(front *mongomodel.Front, now time.Time, reason string) (bool, int) {
	if front == nil {
		return false, 0
	}
	next := buildFrontRailSegments(front.Garrisons)
	railChanged := !frontRailSegmentsEqual(front.RailSegments, next)
	front.RailSegments = next

	garrisons := make(map[string]struct{}, len(front.Garrisons))
	for _, garrison := range front.Garrisons {
		garrisons[garrison.ID] = struct{}{}
	}
	segments := make(map[string]struct{}, len(next))
	for _, segment := range next {
		segments[frontRailPairKey(segment.SourceGarrisonID, segment.TargetGarrisonID)] = struct{}{}
	}

	cancelled := 0
	for i := range front.TradeRoutes {
		route := &front.TradeRoutes[i]
		if route.Status != frontTradeRouteActive {
			continue
		}
		valid := len(route.Waypoints) >= 2
		for waypointIndex := range route.Waypoints {
			if _, ok := garrisons[route.Waypoints[waypointIndex].GarrisonID]; !ok {
				valid = false
				break
			}
			if waypointIndex == 0 {
				continue
			}
			previous := route.Waypoints[waypointIndex-1].GarrisonID
			current := route.Waypoints[waypointIndex].GarrisonID
			if _, ok := segments[frontRailPairKey(previous, current)]; !ok {
				valid = false
				break
			}
		}
		if valid {
			continue
		}
		route.Status = frontTradeRouteCancelled
		route.CancellationReason = reason
		route.SettledAt = timePtr(now)
		cancelled++
	}
	return railChanged, cancelled
}

func buildFrontRailSegments(garrisons []mongomodel.FrontGarrison) []mongomodel.FrontRailSegment {
	ordered := append([]mongomodel.FrontGarrison(nil), garrisons...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	byID := make(map[string]mongomodel.FrontGarrison, len(ordered))
	for _, garrison := range ordered {
		byID[garrison.ID] = garrison
	}

	pairs := make(map[string][2]string)
	for _, source := range ordered {
		neighbors := make([]frontRailNeighbor, 0, len(ordered))
		for _, target := range ordered {
			if source.ID == target.ID {
				continue
			}
			distance := frontGarrisonDistance(source, target)
			if distance > frontRailMaxDistance {
				continue
			}
			neighbors = append(neighbors, frontRailNeighbor{id: target.ID, distance: distance})
		}
		sort.Slice(neighbors, func(i, j int) bool {
			if neighbors[i].distance != neighbors[j].distance {
				return neighbors[i].distance < neighbors[j].distance
			}
			return neighbors[i].id < neighbors[j].id
		})
		for index := 0; index < minFrontInt(frontRailNeighborLimit, len(neighbors)); index++ {
			first, second := frontRailOrderedPair(source.ID, neighbors[index].id)
			pairs[frontRailPairKey(first, second)] = [2]string{first, second}
		}
	}

	segments := make([]mongomodel.FrontRailSegment, 0, len(pairs))
	for _, pair := range pairs {
		source := byID[pair[0]]
		target := byID[pair[1]]
		segments = append(segments, mongomodel.FrontRailSegment{
			ID:               "front_rail_" + source.ID + "__" + target.ID,
			SourceGarrisonID: source.ID,
			TargetGarrisonID: target.ID,
			SourceX:          source.X,
			SourceY:          source.Y,
			TargetX:          target.X,
			TargetY:          target.Y,
			Distance:         frontGarrisonDistance(source, target),
		})
	}
	sort.Slice(segments, func(i, j int) bool { return segments[i].ID < segments[j].ID })
	return segments
}

func frontRailShortestPath(segments []mongomodel.FrontRailSegment, sourceID string, targetID string) []string {
	if sourceID == targetID {
		return []string{sourceID}
	}
	adjacency := make(map[string][]frontRailNeighbor)
	for _, segment := range segments {
		adjacency[segment.SourceGarrisonID] = append(adjacency[segment.SourceGarrisonID], frontRailNeighbor{id: segment.TargetGarrisonID, distance: segment.Distance})
		adjacency[segment.TargetGarrisonID] = append(adjacency[segment.TargetGarrisonID], frontRailNeighbor{id: segment.SourceGarrisonID, distance: segment.Distance})
	}
	if len(adjacency[sourceID]) == 0 || len(adjacency[targetID]) == 0 {
		return nil
	}
	for node := range adjacency {
		sort.Slice(adjacency[node], func(i, j int) bool { return adjacency[node][i].id < adjacency[node][j].id })
	}

	distance := make(map[string]int, len(adjacency))
	previous := make(map[string]string, len(adjacency))
	visited := make(map[string]bool, len(adjacency))
	for node := range adjacency {
		distance[node] = math.MaxInt
	}
	distance[sourceID] = 0
	for len(visited) < len(adjacency) {
		current := ""
		currentDistance := math.MaxInt
		for node, candidateDistance := range distance {
			if visited[node] || candidateDistance > currentDistance {
				continue
			}
			if candidateDistance < currentDistance || current == "" || node < current {
				current = node
				currentDistance = candidateDistance
			}
		}
		if current == "" || currentDistance == math.MaxInt {
			break
		}
		if current == targetID {
			break
		}
		visited[current] = true
		for _, neighbor := range adjacency[current] {
			candidateDistance := currentDistance + neighbor.distance
			knownDistance := distance[neighbor.id]
			if candidateDistance < knownDistance || (candidateDistance == knownDistance && (previous[neighbor.id] == "" || current < previous[neighbor.id])) {
				distance[neighbor.id] = candidateDistance
				previous[neighbor.id] = current
			}
		}
	}
	if distance[targetID] == math.MaxInt {
		return nil
	}
	path := []string{targetID}
	for path[len(path)-1] != sourceID {
		parent := previous[path[len(path)-1]]
		if parent == "" {
			return nil
		}
		path = append(path, parent)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func frontRailSegmentsEqual(left []mongomodel.FrontRailSegment, right []mongomodel.FrontRailSegment) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func frontRailOrderedPair(first string, second string) (string, string) {
	if first < second {
		return first, second
	}
	return second, first
}

func frontRailPairKey(first string, second string) string {
	first, second = frontRailOrderedPair(first, second)
	return first + "\x00" + second
}

func frontGarrisonDistance(first mongomodel.FrontGarrison, second mongomodel.FrontGarrison) int {
	return absFrontInt(first.X-second.X) + absFrontInt(first.Y-second.Y)
}
