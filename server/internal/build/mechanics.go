package build

import (
	"fmt"
	"math"
	"sort"
)

type connectorPosition struct {
	x int
	y int
	z int
}

type worldConnector struct {
	placementIndex int
	placementID    string
	connectorID    string
	kind           string
	direction      string
	capacity       int
}

type connectionKey struct {
	aIndex int
	bIndex int
	kind   string
}

type connectionAccumulator struct {
	aPlacementID  string
	bPlacementID  string
	aConnectorIDs []string
	bConnectorIDs []string
	kind          string
	engagedCount  int
	capacity      int
}

func deriveExactConnections(placements []Placement, catalog PartCatalog) []Connection {
	buckets := make(map[connectorPosition][]worldConnector)
	for placementIndex, placement := range placements {
		part, ok := catalog[placement.PartID]
		if !ok {
			continue
		}
		for connectorIndex, connector := range part.Connectors {
			connectorID := connector.ID
			if connectorID == "" {
				connectorID = fmt.Sprintf("%s-%03d", connector.Kind, connectorIndex+1)
			}
			capacity := connector.CapacityUnits
			if capacity <= 0 {
				capacity = 1
			}
			position, direction := transformConnector(placement, part, connector)
			buckets[position] = append(buckets[position], worldConnector{
				placementIndex: placementIndex,
				placementID:    placement.ID,
				connectorID:    connectorID,
				kind:           connector.Kind,
				direction:      direction,
				capacity:       capacity,
			})
		}
	}

	connections := make(map[connectionKey]*connectionAccumulator)
	for _, endpoints := range buckets {
		sort.Slice(endpoints, func(i, j int) bool {
			if endpoints[i].placementIndex != endpoints[j].placementIndex {
				return endpoints[i].placementIndex < endpoints[j].placementIndex
			}
			return endpoints[i].connectorID < endpoints[j].connectorID
		})
		for i := 0; i < len(endpoints); i++ {
			for j := i + 1; j < len(endpoints); j++ {
				left, right := endpoints[i], endpoints[j]
				if left.placementIndex == right.placementIndex || !directionsOppose(left.direction, right.direction) {
					continue
				}
				kind, compatible := connectionKindFor(left.kind, right.kind)
				if !compatible {
					continue
				}
				if left.placementIndex > right.placementIndex {
					left, right = right, left
				}
				key := connectionKey{aIndex: left.placementIndex, bIndex: right.placementIndex, kind: kind}
				entry := connections[key]
				if entry == nil {
					entry = &connectionAccumulator{aPlacementID: left.placementID, bPlacementID: right.placementID, kind: kind}
					connections[key] = entry
				}
				entry.aConnectorIDs = append(entry.aConnectorIDs, left.connectorID)
				entry.bConnectorIDs = append(entry.bConnectorIDs, right.connectorID)
				entry.engagedCount++
				entry.capacity += min(left.capacity, right.capacity)
			}
		}
	}

	keys := make([]connectionKey, 0, len(connections))
	for key := range connections {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].aIndex != keys[j].aIndex {
			return keys[i].aIndex < keys[j].aIndex
		}
		if keys[i].bIndex != keys[j].bIndex {
			return keys[i].bIndex < keys[j].bIndex
		}
		return keys[i].kind < keys[j].kind
	})
	result := make([]Connection, 0, len(keys))
	for index, key := range keys {
		entry := connections[key]
		result = append(result, Connection{
			ID: fmt.Sprintf("c%02d", index+1), APlacementID: entry.aPlacementID, BPlacementID: entry.bPlacementID,
			AConnectorIDs: entry.aConnectorIDs, BConnectorIDs: entry.bConnectorIDs, Kind: entry.kind,
			EngagedCount: entry.engagedCount, CapacityUnits: entry.capacity,
		})
	}
	return result
}

func transformConnector(placement Placement, part PartSpec, connector PartConnector) (connectorPosition, string) {
	rotation := normalizedRotation(placement.Rotation)
	x, z := rotateXZ(connector.X, connector.Z, rotation)
	sizeX, sizeZ := part.StudsX, part.StudsZ
	if rotation%180 != 0 {
		sizeX, sizeZ = sizeZ, sizeX
	}
	return connectorPosition{
		x: placement.X*20 + sizeX*10 + x,
		y: -(placement.Y+part.PlatesY)*8 + part.OriginYOffsetLDU + connector.Y,
		z: placement.Z*20 + sizeZ*10 + part.OriginCenterZOffsetLDU + z,
	}, rotateDirection(connector.Direction, rotation)
}

func normalizedRotation(rotation int) int {
	return ((rotation % 360) + 360) % 360
}

func rotateXZ(x, z, rotation int) (int, int) {
	switch rotation {
	case 90:
		return -z, x
	case 180:
		return -x, -z
	case 270:
		return z, -x
	default:
		return x, z
	}
}

func rotateDirection(direction string, rotation int) string {
	if direction == "up" || direction == "down" {
		return direction
	}
	order := []string{"north", "east", "south", "west"}
	for index, candidate := range order {
		if candidate == direction {
			return order[(index+rotation/90)%len(order)]
		}
	}
	return direction
}

func directionsOppose(left, right string) bool {
	opposite := map[string]string{"up": "down", "down": "up", "north": "south", "south": "north", "east": "west", "west": "east"}
	return opposite[left] == right
}

func connectionKindFor(left, right string) (string, bool) {
	pair := left + ":" + right
	switch pair {
	case "stud:receptor", "receptor:stud":
		return "stud", true
	case "wheel_pin:wheel_hole", "wheel_hole:wheel_pin":
		return "wheel_pin", true
	case "axle:axle_hole", "axle_hole:axle":
		return "axle", true
	case "pin:pin_hole", "pin_hole:pin":
		return "pin", true
	default:
		return "", false
	}
}

func partIsMechanicallyCertified(part PartSpec) bool {
	certified := part.CertificationLevel == "basic" || part.CertificationLevel == "advanced" || part.CertificationLevel == "certified"
	if !certified || !part.AutoBuildEligible || part.StudsX <= 0 || part.StudsZ <= 0 || part.PlatesY <= 0 ||
		part.Occupancy.Profile == "" || part.Occupancy.StudsX != part.StudsX || part.Occupancy.StudsZ != part.StudsZ ||
		part.Occupancy.PlatesY != part.PlatesY || len(part.Connectors) == 0 {
		return false
	}
	if part.Occupancy.GroundContactProfile != "footprint" && part.Occupancy.GroundContactProfile != "wheel_point" {
		return false
	}
	seen := make(map[string]struct{}, len(part.Connectors))
	for _, connector := range part.Connectors {
		if connector.ID == "" || connector.Kind == "" || connector.CapacityUnits <= 0 || !validConnectorDirection(connector.Direction) {
			return false
		}
		if _, duplicate := seen[connector.ID]; duplicate {
			return false
		}
		seen[connector.ID] = struct{}{}
	}
	return true
}

func validConnectorDirection(direction string) bool {
	switch direction {
	case "up", "down", "north", "east", "south", "west":
		return true
	default:
		return false
	}
}

func supportEngagement(placement Placement, placementsByID map[string]Placement, connections []Connection, catalog PartCatalog) int {
	engaged := 0
	for _, connection := range connections {
		if connection.Kind != "stud" {
			continue
		}
		otherID := ""
		switch placement.ID {
		case connection.APlacementID:
			otherID = connection.BPlacementID
		case connection.BPlacementID:
			otherID = connection.APlacementID
		default:
			continue
		}
		other, ok := placementsByID[otherID]
		if ok && other.Step < placement.Step && topYWith(other, catalog) == placement.Y {
			engaged += connection.EngagedCount
		}
	}
	return engaged
}

func minimumSupportEngagement(placement Placement, catalog PartCatalog) int {
	size := orientedSizeWith(placement, catalog)
	if size.x*size.z >= 4 {
		return 2
	}
	return 1
}

type supportPoint struct {
	x float64
	z float64
}

func validateStepStability(placements []Placement, connections []Connection, catalog PartCatalog, maxStep int) ([]ValidationIssue, int) {
	issues := make([]ValidationIssue, 0)
	minimumMargin := math.Inf(1)
	indexByID := make(map[string]int, len(placements))
	for index, placement := range placements {
		indexByID[placement.ID] = index
	}
	adjacency := make([][]int, len(placements))
	for _, connection := range connections {
		left, leftOK := indexByID[connection.APlacementID]
		right, rightOK := indexByID[connection.BPlacementID]
		if !leftOK || !rightOK {
			continue
		}
		adjacency[left] = append(adjacency[left], right)
		adjacency[right] = append(adjacency[right], left)
	}
	active := make([]bool, len(placements))
	visited := make([]bool, len(placements))
	queue := make([]int, 0, len(placements))
	component := make([]Placement, 0, len(placements))
	for step := 1; step <= maxStep; step++ {
		for index, placement := range placements {
			if placement.Step == step {
				active[index] = true
			}
		}
		for index := range visited {
			visited[index] = false
		}
		stepStable := true
		for start := range placements {
			if !active[start] || visited[start] {
				continue
			}
			queue = append(queue[:0], start)
			visited[start] = true
			component = component[:0]
			for head := 0; head < len(queue); head++ {
				current := queue[head]
				component = append(component, placements[current])
				for _, neighbor := range adjacency[current] {
					if active[neighbor] && !visited[neighbor] {
						visited[neighbor] = true
						queue = append(queue, neighbor)
					}
				}
			}
			stable, margin := componentStabilityMargin(component, catalog)
			if margin < minimumMargin {
				minimumMargin = margin
			}
			if !stable {
				stepStable = false
			}
		}
		if !stepStable {
			issues = append(issues, ValidationIssue{Code: "unstable_step", Message: fmt.Sprintf("第 %d 步存在重心超出实际支撑域的组件", step)})
		}
	}
	if math.IsInf(minimumMargin, 1) {
		return issues, 0
	}
	if math.IsInf(minimumMargin, -1) {
		return issues, -1
	}
	return issues, int(math.Round(minimumMargin * 1000))
}

func componentStabilityMargin(component []Placement, catalog PartCatalog) (bool, float64) {
	if len(component) == 0 {
		return false, math.Inf(-1)
	}
	hasWheelGroundContact := false
	for _, placement := range component {
		if placement.Y == 0 && catalog[placement.PartID].Occupancy.GroundContactProfile == "wheel_point" {
			hasWheelGroundContact = true
			break
		}
	}
	points := make([]supportPoint, 0, len(component)*4)
	weightedX, weightedZ, totalMass := 0.0, 0.0, 0.0
	for _, placement := range component {
		part, ok := catalog[placement.PartID]
		if !ok {
			continue
		}
		size := orientedSizeWith(placement, catalog)
		mass := float64(part.Occupancy.StudsX * part.Occupancy.StudsZ * part.Occupancy.PlatesY)
		if mass <= 0 {
			mass = float64(size.x * size.z * part.PlatesY)
		}
		weightedX += (float64(placement.X) + float64(size.x)/2) * mass
		weightedZ += (float64(placement.Z) + float64(size.z)/2) * mass
		totalMass += mass
		if placement.Y != 0 || (hasWheelGroundContact && part.Occupancy.GroundContactProfile != "wheel_point") {
			continue
		}
		if part.Occupancy.GroundContactProfile == "wheel_point" {
			points = append(points, supportPoint{x: float64(placement.X) + float64(size.x)/2, z: float64(placement.Z) + float64(size.z)/2})
			continue
		}
		points = append(points,
			supportPoint{x: float64(placement.X), z: float64(placement.Z)},
			supportPoint{x: float64(placement.X + size.x), z: float64(placement.Z)},
			supportPoint{x: float64(placement.X + size.x), z: float64(placement.Z + size.z)},
			supportPoint{x: float64(placement.X), z: float64(placement.Z + size.z)},
		)
	}
	if totalMass == 0 || len(points) < 3 {
		return false, math.Inf(-1)
	}
	hull := convexSupportHull(points)
	if len(hull) < 3 {
		return false, math.Inf(-1)
	}
	centerX, centerZ := weightedX/totalMass, weightedZ/totalMass
	margin := math.Inf(1)
	for index, start := range hull {
		end := hull[(index+1)%len(hull)]
		dx, dz := end.x-start.x, end.z-start.z
		cross := dx*(centerZ-start.z) - dz*(centerX-start.x)
		distance := cross / math.Hypot(dx, dz)
		if distance < margin {
			margin = distance
		}
	}
	return margin >= -1e-9, margin
}

func convexSupportHull(points []supportPoint) []supportPoint {
	sorted := append([]supportPoint(nil), points...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].x != sorted[j].x {
			return sorted[i].x < sorted[j].x
		}
		return sorted[i].z < sorted[j].z
	})
	unique := sorted[:0]
	for _, point := range sorted {
		if len(unique) == 0 || unique[len(unique)-1] != point {
			unique = append(unique, point)
		}
	}
	if len(unique) <= 1 {
		return unique
	}
	lower := make([]supportPoint, 0, len(unique))
	for _, point := range unique {
		for len(lower) >= 2 && supportCross(lower[len(lower)-2], lower[len(lower)-1], point) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, point)
	}
	upper := make([]supportPoint, 0, len(unique))
	for index := len(unique) - 1; index >= 0; index-- {
		point := unique[index]
		for len(upper) >= 2 && supportCross(upper[len(upper)-2], upper[len(upper)-1], point) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, point)
	}
	return append(lower[:len(lower)-1], upper[:len(upper)-1]...)
}

func supportCross(origin, left, right supportPoint) float64 {
	return (left.x-origin.x)*(right.z-origin.z) - (left.z-origin.z)*(right.x-origin.x)
}
